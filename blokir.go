package main

import (
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"
)

func (h *Handler) buatBlokir(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SumberDayaID *int    `json:"sumber_daya_id"`
		Tanggal      *string `json:"tanggal"`
		SlotMulai    *int    `json:"slot_mulai"`
		SlotSelesai  *int    `json:"slot_selesai"`
		Keterangan   *string `json:"keterangan"`
	}
	if err := bacaJSON(r, &body); err != nil {
		kirimError(w, 400, "body JSON tidak valid")
		return
	}
	if body.SumberDayaID == nil || body.Tanggal == nil || body.SlotMulai == nil || body.SlotSelesai == nil || body.Keterangan == nil {
		kirimError(w, 400, "kelima field blokir wajib ada")
		return
	}
	tanggal := strings.TrimSpace(*body.Tanggal)
	keterangan := strings.TrimSpace(*body.Keterangan)
	if !tanggalValid(tanggal) || *body.SlotMulai < 1 || *body.SlotSelesai > 8 || *body.SlotMulai > *body.SlotSelesai || utf8.RuneCountInString(keterangan) == 0 || utf8.RuneCountInString(keterangan) > 100 {
		kirimError(w, 400, "tanggal, slot, atau keterangan tidak valid")
		return
	}
	data := BlokirJadwal{SumberDayaID: *body.SumberDayaID, Tanggal: tanggal, SlotMulai: *body.SlotMulai, SlotSelesai: *body.SlotSelesai, Keterangan: keterangan}
	blokir, dibatalkan, err := h.store.tambahBlokir(data)
	if errors.Is(err, ErrSumberDayaTidakValid) {
		kirimError(w, 400, "sumber daya tidak ditemukan")
		return
	}
	if errors.Is(err, ErrBlokirBentrok) {
		kirimError(w, 409, err.Error())
		return
	}
	kirimJSON(w, 201, struct {
		Blokir     BlokirJadwal `json:"blokir"`
		Dibatalkan []int        `json:"dibatalkan"`
	}{Blokir: blokir, Dibatalkan: dibatalkan})
}

func (h *Handler) lihatBlokir(w http.ResponseWriter, r *http.Request) {
	kirimJSON(w, 200, h.store.daftarBlokir())
}

func (h *Handler) hapusBlokir(w http.ResponseWriter, r *http.Request) {
	id, err := bacaID(r)
	if err != nil {
		kirimError(w, 400, err.Error())
		return
	}
	if !h.store.hapusBlokir(id) {
		kirimError(w, 404, "blokir tidak ditemukan")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
