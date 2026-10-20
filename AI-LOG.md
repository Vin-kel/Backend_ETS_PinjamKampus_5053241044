# AI-LOG

## Implementasi REST API B1-B3
- Prompt: Meminta implementasi PinjamKampus Persona E berdasarkan Requirement ETS dan PROSES.md dengan struktur enam file Go.
- Dipakai: `model.go`, `store.go`, `handler.go`, `main.go`, `go.mod`, dan rancangan awal `blokir.go`.
- Perbaikan lanjutan: Validasi dan bentrok belum lengkap pada tahap awal; ditambahkan bertahap dalam permintaan berikutnya.

## B4-B5 Validasi dan bentrok
- Prompt: Melanjutkan implementasi B4 dan B5.
- Dipakai: Validasi JSON dan input di `handler.go`, pemeriksaan kapasitas per slot di `store.go`.
- Perbaikan lanjutan: Fungsi pemeriksaan bentrok dipisahkan agar dapat digunakan pada PUT.

## B6 Ubah dan hapus
- Prompt: Melanjutkan implementasi kode B6.
- Dipakai: Endpoint PUT dan DELETE di `handler.go`, metode ubah dan hapus di `store.go`.
- Perbaikan: Pada PUT, ID peminjaman sendiri dikecualikan dari perhitungan bentrok.

## B7 Permintaan bersamaan
- Prompt: Melanjutkan B7.
- Dipakai: `api_test.go` untuk mengirim 50 request bersamaan.
- Catatan: Mutex di Store sudah ada sebelumnya; tes dibuat untuk memeriksa perilakunya.

## Fitur blokir Persona E
- Prompt: Melanjutkan implementasi fitur khas Persona E.
- Dipakai: `blokir.go`, `handler.go`, `store.go`, dan pengujian pada `api_test.go`.
- Keputusan: Pembuatan blokir dan pembatalan peminjaman dilakukan dalam satu mutex.

## Change Request B — Daftar Bentrok
- Prompt: Mengimplementasikan kode berdasarkan lembar CR-B Daftar Bentrok.
- Dipakai: `BentrokKapasitasError` dan `daftarPenghalang` di `store.go`, respons POST khusus di `handler.go`, tes CR-B di `api_test.go`.
- Keputusan/perbaikan: ID diurutkan ketika mutex masih terkunci; field `bentrok_dengan` hanya muncul pada 409 karena kapasitas POST, bukan error input, blokir, atau PUT.

## Verifikasi
- Pengujian belum dijalankan oleh AI karena instalasi Go tidak tersedia pada lingkungan eksekusi.
- Jalankan sendiri `go test -race ./...`, `go vet ./...`, dan `go build ./...`, lalu catat hasil nyata serta perubahan yang Anda lakukan.
