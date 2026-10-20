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

## 4. Bug atau kendala 
### Bug 1
- Gejala: Saat mencoba menjalankan go run ., muncul pesan runtime.main_main·f: function main is undeclared in the main package. Server belum bisa dijalankan.
- Cara menemukan: Pesan error muncul di terminal ketika menjalankan proyek. Saya kemudian memeriksa direktori kerja, file main.go, dan keberadaan fungsi func main().
Penyebab dan solusi: Pesan tersebut berarti Go tidak menemukan fungsi entry point
- Penyebab dan solusi: Pesan tersebut berarti Go tidak menemukan fungsi entry point main() pada paket yang sedang dijalankan. Solusinya adalah memastikan main.go berada di direktori proyek yang benar, menggunakan package main, dan memiliki func main() yang menjalankan http.ListenAndServe. Penyebab spesifik di komputer saya dan hasil perbaikannya masih perlu dikonfirmasi setelah pengecekan.

### Bug 2
- Gejala/kendala: Pada rancangan awal pemeriksaan bentrok, ada risiko peminjaman yang sedang diubah melalui PUT ikut dihitung sebagai peminjaman lain. Akibatnya perubahan pada slot yang sama bisa ditolak 409 meskipun sebenarnya tidak melewati stok.
- Cara menemukan: Saya menemukan risiko ini saat meninjau logika pemeriksaan bentrok untuk B6 dan membandingkannya dengan aturan PUT dalam requirement. Ini merupakan potensi bug dari peninjauan kode, bukan error yang sudah saya buktikan muncul saat menjalankan program.
- Penyebab dan solusi: Perhitungan kapasitas semula memeriksa seluruh peminjaman aktif. Saya menambahkan parameter kecualiID pada fungsi adaBentrok() agar peminjaman dengan ID yang sedang diubah tidak dihitung dua kali. Setelah itu, PUT tetap memeriksa peminjaman lain dan menolak perubahan yang benar-benar bentrok.

## 5. Change request
Pada Change Request B, tim frontend meminta agar respons 409 dari POST /peminjaman karena slot atau kapasitas penuh tidak hanya berisi pesan error, tetapi juga field bentrok_dengan yang menunjukkan ID peminjaman penghalang secara urut naik. Field tersebut tidak boleh muncul pada error lain atau pada respons PUT. Untuk Persona E, jika penolakan disebabkan oleh blokir jadwal, field tersebut tidak diwajibkan.

Pengubahan store.go dengan menambahkan daftarPenghalang() untuk mengumpulkan ID peminjaman aktif yang menggunakan sumber daya dan tanggal sama serta mempunyai rentang slot beririsan. Saya juga membuat BentrokKapasitasError untuk membawa daftar ID tersebut. Di handler.go, respons POST /peminjaman diubah agar menampilkan error dan bentrok_dengan khusus saat kapasitas penuh. Pengujian kasus tersebut ditambahkan ke api_test.go.

Keputusan desain adalah mengumpulkan ID penghalang ketika sync.Mutex pada Store masih terkunci, lalu membawanya ke handler melalui tipe error khusus. Dengan begitu daftar ID berasal dari kondisi data yang sama dengan saat bentrok diperiksa, dan respons error untuk PUT maupun blokir tidak ikut berubah.
