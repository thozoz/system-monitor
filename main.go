package main

import (
	"fmt"
	"net/http"

	"github.com/shirou/gopsutil/v3/mem"
)

func statusHandler(w http.ResponseWriter, r *http.Request) {
	// RAM (Virtual Memory) bilgilerini okuyoruz
	// v değişkeni pointer, err ise hata kontrolü için
	v, err := mem.VirtualMemory()

	if err != nil {
		fmt.Fprintf(w, "RAM bilgisi okunamadi: %v", err)
		return
	}

	// Bilgiler byte cinsinden, GB'a çevirmek için 1024'e arka arkaya 3 kere bölüyoruz.(byte->kilobayt->megabayt->gigabyte)
	toplamGB := v.Total / 1024 / 1024 / 1024
	kullanilanGB := v.Used / 1024 / 1024 / 1024

	// Tarayıcıya düz metin olarak bilgileri yolla
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "--- SISTEM RAM DURUMU ---\n")
	fmt.Fprintf(w, "Toplam RAM: %v GB\n", toplamGB)
	fmt.Fprintf(w, "Kullanilan RAM: %v GB\n", kullanilanGB)

	fmt.Fprintf(w, "Kullanim Orani: %%%.2f\n", v.UsedPercent)
}

func main() {
	http.HandleFunc("/api/status", statusHandler)

	fmt.Println("Sunucu 8080 portunda calismaya basladi. http://localhost:8080/api/status adresine gidebilirsin.")

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("HATA:", err)
	}
}
