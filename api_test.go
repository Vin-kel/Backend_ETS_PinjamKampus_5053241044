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
