package main

import (
	"errors"
	"sort"
	"sync"
)

var (
	ErrSumberDayaTidakValid = errors.New("sumber daya tidak ada atau jumlah melebihi stok")
	ErrBentrok              = errors.New("kapasitas slot tidak mencukupi")
)

type Store struct {
	mu               sync.Mutex
	sumberDaya       map[int]SumberDaya
	peminjaman       map[int]Peminjaman
	blokir           map[int]BlokirJadwal
	nextSumberDayaID int
	nextPeminjamanID int
	nextBlokirID     int
}

func NewStore() *Store {
	return &Store{
		sumberDaya:       make(map[int]SumberDaya),
		peminjaman:       make(map[int]Peminjaman),
		blokir:           make(map[int]BlokirJadwal),
		nextSumberDayaID: 1,
		nextPeminjamanID: 1,
		nextBlokirID:     1,
	}
}

func (s *Store) tambahSumberDaya(data SumberDaya) SumberDaya {
	s.mu.Lock()
	defer s.mu.Unlock()
	data.ID = s.nextSumberDayaID
	s.nextSumberDayaID++
	s.sumberDaya[data.ID] = data
	return data
}

func (s *Store) daftarSumberDaya() []SumberDaya {
	s.mu.Lock()
	defer s.mu.Unlock()
	hasil := make([]SumberDaya, 0, len(s.sumberDaya))
	for _, data := range s.sumberDaya {
		hasil = append(hasil, data)
	}
	sort.Slice(hasil, func(i, j int) bool { return hasil[i].ID < hasil[j].ID })
	return hasil
}

func (s *Store) cariSumberDaya(id int) (SumberDaya, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, ada := s.sumberDaya[id]
	return data, ada
}

func (s *Store) tambahPeminjaman(data Peminjaman) (Peminjaman, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sumber, ada := s.sumberDaya[data.SumberDayaID]
	if !ada || data.Jumlah < 1 || data.Jumlah > sumber.Stok {
		return Peminjaman{}, ErrSumberDayaTidakValid
	}
	// B5: setiap slot harus memiliki kapasitas yang cukup.
	// Pemeriksaan dan penyimpanan terjadi dalam mutex yang sama.
	for slot := data.SlotMulai; slot <= data.SlotSelesai; slot++ {
		terpakai := 0
		for _, lama := range s.peminjaman {
			if lama.Status != "aktif" || lama.SumberDayaID != data.SumberDayaID || lama.Tanggal != data.Tanggal {
				continue
			}
			if lama.SlotMulai <= slot && slot <= lama.SlotSelesai {
				terpakai += lama.Jumlah
			}
		}
		if terpakai+data.Jumlah > sumber.Stok {
			return Peminjaman{}, ErrBentrok
		}
	}
	data.ID = s.nextPeminjamanID
	s.nextPeminjamanID++
	data.Status = "aktif"
	s.peminjaman[data.ID] = data
	return data, nil
}

func (s *Store) daftarPeminjaman(sumberID *int, tanggal, status string) []Peminjaman {
	s.mu.Lock()
	defer s.mu.Unlock()
	hasil := make([]Peminjaman, 0)
	for _, data := range s.peminjaman {
		if sumberID != nil && data.SumberDayaID != *sumberID {
			continue
		}
		if tanggal != "" && data.Tanggal != tanggal {
			continue
		}
		if status != "" && data.Status != status {
			continue
		}
		hasil = append(hasil, data)
	}
	sort.Slice(hasil, func(i, j int) bool { return hasil[i].ID < hasil[j].ID })
	return hasil
}

func (s *Store) cariPeminjaman(id int) (Peminjaman, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, ada := s.peminjaman[id]
	return data, ada
}
