# PROSES

## 1. Persona dan masalahnya
Pengguna dari fitur ini adalah asisten laboratorium yang bertugas mengatur jadwal penggunaan lab. Masalahnya, laboratorium terkadang digunakan untuk praktikum atau perbaikan sehingga tidak bisa dipinjam pada waktu tertentu. Karena itu, diperlukan fitur blokir jadwal agar peminjaman baru pada slot yang diblokir ditolak dan peminjaman yang sudah ada pada jadwal tersebut dibatalkan.

## 2. Rancangan struktur data
Tiga Struct utama yang dirancang adalah SumberDaya, Peminjaman, dan BlokirJadwal.SumberDaya menyimpan ID, nama, jenis, dan stok. Peminjaman menyimpan ID, sumber daya yang dipinjam, nama peminjam, tanggal, rentang slot, jumlah, dan status.

Ekslusif untuk persona E, persona E menggunakan BlokirJadwal yang berisi ID, sumber daya, tanggal, slot mulai, slot selesai, dan keterangan. Data ini digunakan untuk memeriksa apakah suatu jadwal sedang diblokir. Ketika blokir dibuat, peminjaman aktif yang terkena blokir akan diubah statusnya menjadi dibatalkan, tetapi datanya tetap disimpan.

Ketiga jenis data tersebut akan disimpan menggunakan map dengan ID sebagai key di dalam struct Store. map dipilih agar data mudah dicari, diperbarui, dan dihapus berdasarkan ID. Setiap jenis data memiliki penghitung ID secara individu.

Penggunakan sync.Mutex adalah untuk melindungi data ketika beberapa request diproses bersamaan. Untuk endpoint yang menampilkan daftar, data dari map akan dikumpulkan ke dalam slice lalu diurutkan berdasarkan ID sebelum dikirim sebagai JSON.

## 3. Satu keputusan desain
Keputusan saya adalah untuk penyimpanan data blokir jadwal secara terpisah dari data peminjaman, tetapi keduanya tetap dikelola dalam satu Store. Alternatif yang dipertimbangkan adalah menyimpan blokir sebagai bagian dari data peminjaman.

Saya memutuskan pemisahan tersebut karena blokir jadwal bukan peminjaman biasa dan memiliki informasi tambahan seperti keterangan. Dengan rancangan ini, saat blokir dibuat, sistem dapat mencari peminjaman yang terkena blokir dan mengubah statusnya menjadi dibatalkan tanpa menghapus data peminjaman tersebut.

## 4. Bug atau kendala (minimal dua)
### Bug 1
- Gejala:
- Cara menemukan:
- Penyebab dan solusi:

### Bug 2
- Gejala:
- Cara menemukan:
- Penyebab dan solusi:

## 5. Change request
(Apa yang diminta, bagian kode mana yang kamu ubah, dan satu keputusan desain yang kamu ambil.)
