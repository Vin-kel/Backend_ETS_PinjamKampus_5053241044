package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type Handler struct{ store *Store }

// NewHandler selalu membuat store baru, sesuai kontrak pengujian ETS.
func NewHandler() http.Handler {
	h := &Handler{store: NewStore()}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /sumber-daya", h.buatSumberDaya)
	mux.HandleFunc("GET /sumber-daya", h.lihatSumberDaya)
	mux.HandleFunc("GET /sumber-daya/{id}", h.detailSumberDaya)
	mux.HandleFunc("POST /peminjaman", h.buatPeminjaman)
	mux.HandleFunc("GET /peminjaman", h.lihatPeminjaman)
	mux.HandleFunc("GET /peminjaman/{id}", h.detailPeminjaman)
	mux.HandleFunc("PUT /peminjaman/{id}", h.ubahPeminjaman)
	mux.HandleFunc("DELETE /peminjaman/{id}", h.hapusPeminjaman)
	return mux
}

func kirimJSON(w http.ResponseWriter, kode int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(kode)
	_ = json.NewEncoder(w).Encode(data)
}

func kirimError(w http.ResponseWriter, kode int, pesan string) {
	kirimJSON(w, kode, map[string]string{"error": pesan})
}

// Decode satu objek JSON; field tak dikenal dan JSON tambahan dianggap salah input.
func bacaJSON(r *http.Request, tujuan any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(tujuan); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return errors.New("body hanya boleh memuat satu objek JSON")
	}
	return nil
}

func bacaID(r *http.Request) (int, error) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 1 {
		return 0, errors.New("id harus berupa angka positif")
	}
	return id, nil
}

func tanggalValid(tanggal string) bool {
	if len(tanggal) != 10 {
		return false
	}
	t, err := time.Parse("2006-01-02", tanggal)
	return err == nil && t.Format("2006-01-02") == tanggal
}

func (h *Handler) buatSumberDaya(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Nama  *string `json:"nama"`
		Jenis *string `json:"jenis"`
		Stok  *int    `json:"stok"`
	}
	if err := bacaJSON(r, &data); err != nil {
		kirimError(w, 400, "body JSON tidak valid")
		return
	}
	if data.Nama == nil || data.Jenis == nil || data.Stok == nil {
		kirimError(w, 400, "nama, jenis, dan stok wajib ada")
		return
	}
	nama, jenis := strings.TrimSpace(*data.Nama), strings.TrimSpace(*data.Jenis)
	if utf8.RuneCountInString(nama) == 0 || utf8.RuneCountInString(nama) > 50 || (jenis != "ruang" && jenis != "alat") || *data.Stok < 1 {
		kirimError(w, 400, "nama, jenis, atau stok tidak valid")
		return
	}
	hasil := h.store.tambahSumberDaya(SumberDaya{Nama: nama, Jenis: jenis, Stok: *data.Stok})
	kirimJSON(w, 201, hasil)
}

func (h *Handler) lihatSumberDaya(w http.ResponseWriter, r *http.Request) {
	kirimJSON(w, 200, h.store.daftarSumberDaya())
}

func (h *Handler) detailSumberDaya(w http.ResponseWriter, r *http.Request) {
	id, err := bacaID(r)
	if err != nil {
		kirimError(w, 400, err.Error())
		return
	}
	data, ada := h.store.cariSumberDaya(id)
	if !ada {
		kirimError(w, 404, "sumber daya tidak ditemukan")
		return
	}
	kirimJSON(w, 200, data)
}

func (h *Handler) buatPeminjaman(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SumberDayaID *int    `json:"sumber_daya_id"`
		Peminjam     *string `json:"peminjam"`
		Tanggal      *string `json:"tanggal"`
		SlotMulai    *int    `json:"slot_mulai"`
		SlotSelesai  *int    `json:"slot_selesai"`
		Jumlah       *int    `json:"jumlah"`
	}
	if err := bacaJSON(r, &body); err != nil {
		kirimError(w, 400, "body JSON tidak valid")
		return
	}
	if body.SumberDayaID == nil || body.Peminjam == nil || body.Tanggal == nil || body.SlotMulai == nil || body.SlotSelesai == nil || body.Jumlah == nil {
		kirimError(w, 400, "keenam field peminjaman wajib ada")
		return
	}
	peminjam := strings.TrimSpace(*body.Peminjam)
	tanggal := strings.TrimSpace(*body.Tanggal)
	if utf8.RuneCountInString(peminjam) == 0 || utf8.RuneCountInString(peminjam) > 50 || !tanggalValid(tanggal) || *body.SlotMulai < 1 || *body.SlotSelesai > 8 || *body.SlotMulai > *body.SlotSelesai || *body.Jumlah < 1 {
		kirimError(w, 400, "data peminjaman tidak valid")
		return
	}
	data := Peminjaman{SumberDayaID: *body.SumberDayaID, Peminjam: peminjam, Tanggal: tanggal, SlotMulai: *body.SlotMulai, SlotSelesai: *body.SlotSelesai, Jumlah: *body.Jumlah}
	hasil, err := h.store.tambahPeminjaman(data)
	if errors.Is(err, ErrSumberDayaTidakValid) {
		kirimError(w, 400, err.Error())
		return
	}
	if errors.Is(err, ErrBentrok) {
		kirimError(w, 409, err.Error())
		return
	}
	kirimJSON(w, 201, hasil)
}

func (h *Handler) lihatPeminjaman(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	var sumberID *int
	if query.Has("sumber_daya_id") {
		id, err := strconv.Atoi(query.Get("sumber_daya_id"))
		if err != nil {
			kirimError(w, 400, "filter sumber_daya_id harus angka")
			return
		}
		sumberID = &id
	}
	tanggal := strings.TrimSpace(query.Get("tanggal"))
	status := strings.TrimSpace(query.Get("status"))
	kirimJSON(w, 200, h.store.daftarPeminjaman(sumberID, tanggal, status))
}

func (h *Handler) detailPeminjaman(w http.ResponseWriter, r *http.Request) {
	id, err := bacaID(r)
	if err != nil {
		kirimError(w, 400, err.Error())
		return
	}
	data, ada := h.store.cariPeminjaman(id)
	if !ada {
		kirimError(w, 404, "peminjaman tidak ditemukan")
		return
	}
	kirimJSON(w, 200, data)
}

func (h *Handler) ubahPeminjaman(w http.ResponseWriter, r *http.Request) {
	// Urutan: ID (400), keberadaan peminjaman (404), body (400), bentrok (409).
	id, err := bacaID(r)
	if err != nil {
		kirimError(w, 400, err.Error())
		return
	}
	if _, ada := h.store.cariPeminjaman(id); !ada {
		kirimError(w, 404, "peminjaman tidak ditemukan")
		return
	}
	var body struct {
		SumberDayaID *int    `json:"sumber_daya_id"`
		Peminjam     *string `json:"peminjam"`
		Tanggal      *string `json:"tanggal"`
		SlotMulai    *int    `json:"slot_mulai"`
		SlotSelesai  *int    `json:"slot_selesai"`
		Jumlah       *int    `json:"jumlah"`
	}
	if err := bacaJSON(r, &body); err != nil {
		kirimError(w, 400, "body JSON tidak valid")
		return
	}
	if body.SumberDayaID == nil || body.Peminjam == nil || body.Tanggal == nil || body.SlotMulai == nil || body.SlotSelesai == nil || body.Jumlah == nil {
		kirimError(w, 400, "keenam field peminjaman wajib ada")
		return
	}
	peminjam := strings.TrimSpace(*body.Peminjam)
	tanggal := strings.TrimSpace(*body.Tanggal)
	if utf8.RuneCountInString(peminjam) == 0 || utf8.RuneCountInString(peminjam) > 50 || !tanggalValid(tanggal) || *body.SlotMulai < 1 || *body.SlotSelesai > 8 || *body.SlotMulai > *body.SlotSelesai || *body.Jumlah < 1 {
		kirimError(w, 400, "data peminjaman tidak valid")
		return
	}
	data := Peminjaman{SumberDayaID: *body.SumberDayaID, Peminjam: peminjam, Tanggal: tanggal, SlotMulai: *body.SlotMulai, SlotSelesai: *body.SlotSelesai, Jumlah: *body.Jumlah}
	hasil, err := h.store.ubahPeminjaman(id, data)
	if errors.Is(err, ErrPeminjamanTidakAda) {
		kirimError(w, 404, err.Error())
		return
	}
	if errors.Is(err, ErrSumberDayaTidakValid) {
		kirimError(w, 400, err.Error())
		return
	}
	if errors.Is(err, ErrBentrok) {
		kirimError(w, 409, err.Error())
		return
	}
	kirimJSON(w, 200, hasil)
}

func (h *Handler) hapusPeminjaman(w http.ResponseWriter, r *http.Request) {
	id, err := bacaID(r)
	if err != nil {
		kirimError(w, 400, err.Error())
		return
	}
	if !h.store.hapusPeminjaman(id) {
		kirimError(w, 404, "peminjaman tidak ditemukan")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
