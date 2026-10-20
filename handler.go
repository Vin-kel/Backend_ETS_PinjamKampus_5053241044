package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestB7PeminjamanBersamaan(t *testing.T) {
	handler := NewHandler()

	// Hanya ada satu unit yang dapat digunakan pada setiap slot.
	buatSumber := httptest.NewRequest(http.MethodPost, "/sumber-daya", strings.NewReader(`{"nama":"Lab PWEB","jenis":"ruang","stok":1}`))
	resSumber := httptest.NewRecorder()
	handler.ServeHTTP(resSumber, buatSumber)
	if resSumber.Code != http.StatusCreated {
		t.Fatalf("membuat sumber daya: status %d, body %s", resSumber.Code, resSumber.Body.String())
	}

	const totalRequest = 50
	var wg sync.WaitGroup
	mulai := make(chan struct{})
	status := make(chan int, totalRequest)

	for i := 0; i < totalRequest; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			body := fmt.Sprintf(`{"sumber_daya_id":1,"peminjam":"Mahasiswa %d","tanggal":"2026-11-02","slot_mulai":3,"slot_selesai":4,"jumlah":1}`, n)
			req := httptest.NewRequest(http.MethodPost, "/peminjaman", strings.NewReader(body))
			res := httptest.NewRecorder()
			<-mulai // Semua goroutine mulai mengirim setelah kanal dibuka.
			handler.ServeHTTP(res, req)
			status <- res.Code
		}(i)
	}
	close(mulai)
	wg.Wait()
	close(status)

	berhasil, bentrok := 0, 0
	for kode := range status {
		switch kode {
		case http.StatusCreated:
			berhasil++
		case http.StatusConflict:
			bentrok++
		default:
			t.Errorf("status tak terduga: %d", kode)
		}
	}
	if berhasil != 1 || bentrok != totalRequest-1 {
		t.Errorf("ingin 1 sukses dan %d bentrok, mendapat %d sukses dan %d bentrok", totalRequest-1, berhasil, bentrok)
	}

	// Cek kondisi akhir, bukan hanya kode HTTP.
	resDaftar := httptest.NewRecorder()
	handler.ServeHTTP(resDaftar, httptest.NewRequest(http.MethodGet, "/peminjaman", nil))
	if resDaftar.Code != http.StatusOK {
		t.Fatalf("GET peminjaman: status %d", resDaftar.Code)
	}
	var daftar []Peminjaman
	if err := json.Unmarshal(resDaftar.Body.Bytes(), &daftar); err != nil {
		t.Fatalf("decode daftar: %v", err)
	}
	if len(daftar) != 1 {
		t.Errorf("harus tersimpan satu peminjaman, ditemukan %d", len(daftar))
	}
}

func TestPersonaEBlokirMembatalkanDanMenolak(t *testing.T) {
	h := NewHandler()
	kirim := func(method, url, body string, kode int) []byte {
		t.Helper()
		req := httptest.NewRequest(method, url, strings.NewReader(body))
		res := httptest.NewRecorder()
		h.ServeHTTP(res, req)
		if res.Code != kode {
			t.Fatalf("%s %s: status %d, ingin %d; body: %s", method, url, res.Code, kode, res.Body.String())
		}
		return res.Body.Bytes()
	}
	kirim("POST", "/sumber-daya", `{"nama":"Lab","jenis":"ruang","stok":2}`, 201)
	kirim("POST", "/peminjaman", `{"sumber_daya_id":1,"peminjam":"Rina","tanggal":"2026-11-02","slot_mulai":2,"slot_selesai":3,"jumlah":1}`, 201)
	kirim("POST", "/peminjaman", `{"sumber_daya_id":1,"peminjam":"Budi","tanggal":"2026-11-02","slot_mulai":3,"slot_selesai":4,"jumlah":1}`, 201)
	res := kirim("POST", "/blokir", `{"sumber_daya_id":1,"tanggal":"2026-11-02","slot_mulai":3,"slot_selesai":3,"keterangan":"Praktikum"}`, 201)
	var hasil struct {
		Blokir     BlokirJadwal `json:"blokir"`
		Dibatalkan []int        `json:"dibatalkan"`
	}
	if err := json.Unmarshal(res, &hasil); err != nil {
		t.Fatal(err)
	}
	if hasil.Blokir.ID != 1 || len(hasil.Dibatalkan) != 2 || hasil.Dibatalkan[0] != 1 || hasil.Dibatalkan[1] != 2 {
		t.Fatalf("hasil blokir tidak sesuai: %+v", hasil)
	}
	res = kirim("GET", "/peminjaman/1", "", 200)
	var p Peminjaman
	if err := json.Unmarshal(res, &p); err != nil {
		t.Fatal(err)
	}
	if p.Status != "dibatalkan" {
		t.Fatalf("status harus dibatalkan, mendapat %s", p.Status)
	}
	// Kapasitas sudah bebas, tetapi blokir tetap mencegah peminjaman baru dan PUT.
	kirim("POST", "/peminjaman", `{"sumber_daya_id":1,"peminjam":"Ani","tanggal":"2026-11-02","slot_mulai":3,"slot_selesai":4,"jumlah":1}`, 409)
	kirim("PUT", "/peminjaman/1", `{"sumber_daya_id":1,"peminjam":"Rina","tanggal":"2026-11-02","slot_mulai":3,"slot_selesai":3,"jumlah":1}`, 409)
	kirim("POST", "/blokir", `{"sumber_daya_id":1,"tanggal":"2026-11-02","slot_mulai":2,"slot_selesai":3,"keterangan":"Servis"}`, 409)
	kirim("DELETE", "/blokir/1", "", 204)
	// Menghapus blokir tidak memulihkan status peminjaman yang dibatalkan.
	res = kirim("GET", "/peminjaman/1", "", 200)
	if err := json.Unmarshal(res, &p); err != nil {
		t.Fatal(err)
	}
	if p.Status != "dibatalkan" {
		t.Fatalf("hapus blokir memulihkan status: %s", p.Status)
	}
	kirim("POST", "/peminjaman", `{"sumber_daya_id":1,"peminjam":"Ani","tanggal":"2026-11-02","slot_mulai":3,"slot_selesai":4,"jumlah":1}`, 201)
}

func TestCRBDaftarBentrok(t *testing.T) {
	h := NewHandler()
	kirim := func(method, url, body string, status int) map[string]any {
		t.Helper()
		res := httptest.NewRecorder()
		h.ServeHTTP(res, httptest.NewRequest(method, url, strings.NewReader(body)))
		if res.Code != status {
			t.Fatalf("status %d, ingin %d: %s", res.Code, status, res.Body.String())
		}
		var hasil map[string]any
		if err := json.Unmarshal(res.Body.Bytes(), &hasil); err != nil {
			t.Fatal(err)
		}
		return hasil
	}
	kirim("POST", "/sumber-daya", `{"nama":"Lab","jenis":"ruang","stok":1}`, 201)
	kirim("POST", "/peminjaman", `{"sumber_daya_id":1,"peminjam":"Rina","tanggal":"2026-11-02","slot_mulai":2,"slot_selesai":3,"jumlah":1}`, 201)
	kirim("POST", "/peminjaman", `{"sumber_daya_id":1,"peminjam":"Budi","tanggal":"2026-11-02","slot_mulai":5,"slot_selesai":6,"jumlah":1}`, 201)
	hasil := kirim("POST", "/peminjaman", `{"sumber_daya_id":1,"peminjam":"Ani","tanggal":"2026-11-02","slot_mulai":3,"slot_selesai":5,"jumlah":1}`, 409)
	ids, ok := hasil["bentrok_dengan"].([]any)
	if !ok || len(ids) != 2 || ids[0] != float64(1) || ids[1] != float64(2) {
		t.Fatalf("bentrok_dengan salah: %v", hasil)
	}
	if hasil["error"] == nil {
		t.Fatalf("field error hilang: %v", hasil)
	}
	hasil = kirim("POST", "/peminjaman", `{"sumber_daya_id":1,"peminjam":"Ani","tanggal":"2026-11-02","slot_mulai":1,"slot_selesai":2,"jumlah":1}`, 409)
	ids, ok = hasil["bentrok_dengan"].([]any)
	if !ok || len(ids) != 1 || ids[0] != float64(1) {
		t.Fatalf("harus hanya [1]: %v", hasil)
	}
	hasil = kirim("POST", "/peminjaman", `{"sumber_daya_id":1,"peminjam":"","tanggal":"2026-11-02","slot_mulai":3,"slot_selesai":5,"jumlah":1}`, 400)
	if _, ada := hasil["bentrok_dengan"]; ada {
		t.Fatalf("400 tidak boleh mengandung bentrok_dengan")
	}
	kirim("POST", "/blokir", `{"sumber_daya_id":1,"tanggal":"2026-11-03","slot_mulai":3,"slot_selesai":4,"keterangan":"Servis"}`, 201)
	hasil = kirim("POST", "/peminjaman", `{"sumber_daya_id":1,"peminjam":"Ani","tanggal":"2026-11-03","slot_mulai":3,"slot_selesai":4,"jumlah":1}`, 409)
	if _, ada := hasil["bentrok_dengan"]; ada {
		t.Fatalf("409 karena blokir tidak perlu field bentrok_dengan")
	}
}
