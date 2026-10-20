package main

import (
	"errors"
	"sort"
	"sync"
)

var (
	ErrSumberDayaTidakValid = errors.New("sumber daya tidak ada atau jumlah melebihi stok")
	ErrBentrok              = errors.New("kapasitas slot tidak mencukupi")
	ErrPeminjamanTidakAda   = errors.New("peminjaman tidak ditemukan")
	ErrBlokirBentrok        = errors.New("jadwal sudah diblokir")
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
	if s.adaBlokir(data.SumberDayaID, data.Tanggal, data.SlotMulai, data.SlotSelesai) {
		return Peminjaman{}, ErrBlokirBentrok
	}
	if s.adaBentrok(data, sumber.Stok, 0) {
		return Peminjaman{}, ErrBentrok
	}
	data.ID = s.nextPeminjamanID
	s.nextPeminjamanID++
	data.Status = "aktif"
	s.peminjaman[data.ID] = data
	return data, nil
}

// Dipanggil hanya ketika mutex Store sedang dikunci.
// kecualiID = 0 untuk POST; pada PUT, data lama diabaikan dari perhitungan.
func (s *Store) adaBentrok(data Peminjaman, stok, kecualiID int) bool {
	for slot := data.SlotMulai; slot <= data.SlotSelesai; slot++ {
		terpakai := 0
		for _, lama := range s.peminjaman {
			if lama.ID == kecualiID || lama.Status != "aktif" || lama.SumberDayaID != data.SumberDayaID || lama.Tanggal != data.Tanggal {
				continue
			}
			if lama.SlotMulai <= slot && slot <= lama.SlotSelesai {
				terpakai += lama.Jumlah
			}
		}
		if terpakai+data.Jumlah > stok {
			return true
		}
	}
	return false
}

func (s *Store) ubahPeminjaman(id int, data Peminjaman) (Peminjaman, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	lama, ada := s.peminjaman[id]
	if !ada {
		return Peminjaman{}, ErrPeminjamanTidakAda
	}
	sumber, ada := s.sumberDaya[data.SumberDayaID]
	if !ada || data.Jumlah < 1 || data.Jumlah > sumber.Stok {
		return Peminjaman{}, ErrSumberDayaTidakValid
	}
	if s.adaBlokir(data.SumberDayaID, data.Tanggal, data.SlotMulai, data.SlotSelesai) {
		return Peminjaman{}, ErrBlokirBentrok
	}
	if s.adaBentrok(data, sumber.Stok, id) {
		return Peminjaman{}, ErrBentrok
	}
	data.ID = id
	data.Status = lama.Status
	s.peminjaman[id] = data
	return data, nil
}

func (s *Store) hapusPeminjaman(id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ada := s.peminjaman[id]; !ada {
		return false
	}
	delete(s.peminjaman, id)
	return true
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

// Semua pemanggilan adaBlokir berlangsung saat mutex sudah dikunci.
func (s *Store) adaBlokir(sumberID int, tanggal string, mulai, selesai int) bool {
	for _, b := range s.blokir {
		if b.SumberDayaID == sumberID && b.Tanggal == tanggal && mulai <= b.SlotSelesai && b.SlotMulai <= selesai {
			return true
		}
	}
	return false
}

func (s *Store) tambahBlokir(data BlokirJadwal) (BlokirJadwal, []int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ada := s.sumberDaya[data.SumberDayaID]; !ada {
		return BlokirJadwal{}, nil, ErrSumberDayaTidakValid
	}
	if s.adaBlokir(data.SumberDayaID, data.Tanggal, data.SlotMulai, data.SlotSelesai) {
		return BlokirJadwal{}, nil, ErrBlokirBentrok
	}
	data.ID = s.nextBlokirID
	s.nextBlokirID++
	s.blokir[data.ID] = data
	dibatalkan := make([]int, 0)
	for id, p := range s.peminjaman {
		if p.Status != "aktif" || p.SumberDayaID != data.SumberDayaID || p.Tanggal != data.Tanggal {
			continue
		}
		if p.SlotMulai <= data.SlotSelesai && data.SlotMulai <= p.SlotSelesai {
			p.Status = "dibatalkan"
			s.peminjaman[id] = p
			dibatalkan = append(dibatalkan, id)
		}
	}
	sort.Ints(dibatalkan)
	return data, dibatalkan, nil
}

func (s *Store) daftarBlokir() []BlokirJadwal {
	s.mu.Lock()
	defer s.mu.Unlock()
	hasil := make([]BlokirJadwal, 0, len(s.blokir))
	for _, data := range s.blokir {
		hasil = append(hasil, data)
	}
	sort.Slice(hasil, func(i, j int) bool { return hasil[i].ID < hasil[j].ID })
	return hasil
}

func (s *Store) hapusBlokir(id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ada := s.blokir[id]; !ada {
		return false
	}
	delete(s.blokir, id)
	return true
}
