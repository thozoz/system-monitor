package main

import (
	"encoding/json"
	"fmt"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"net/http"
)

// json kalıbımız(struct)
type SystemInfo struct {
	TotalRAM    float32 `json:"toplam_ram_gb"`
	UsedRAM     float32 `json:"kullanilan_ram_gb"`
	UsedPercent float32 `json:"ram_kullanim_orani"`

	TotalSwap   float32 `json:"toplam_swap_gb"`
	UsedSwap    float32 `json:"kullanilan_swap_gb"`
	SwapPercent float32 `json:"swap_kullanim_orani"`

	CPUPercent  float32 `json:"cpu_kullanim_orani"`
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	// ram bilgileri
	v, err := mem.VirtualMemory()
	if err != nil {
		http.Error(w, "RAM bilgisi okunamadi", http.StatusInternalServerError)
		return
	}

	// swap alanı bilgileri
	// s değişkeni bize swap hakkında struct döndürür
	s, err := mem.SwapMemory()
	if err != nil {
		http.Error(w, "Swap bilgisi okunamadi", http.StatusInternalServerError)
		return
	}

	// CPU kullanım bilgileri
	// ilk parametre ne kadar sürelik bir ölçüm yapacağı (0 anlık demektir)
	// ikinci parametre (false) olansa çekirdeklerin ayrı ayrı kullanımı yerine hepsinin ortalamasını verir
	c, err := cpu.Percent(0, false)
	
	// c değişkeni bir dizi olarak döner, o yüzden ilk elemanını [0] alıyoruz
	cpuKullanimi := float32(0)
	if err == nil && len(c) > 0 {
		cpuKullanimi = float32(c[0])
	}

	// kalıp (struct) içini ram ve swap bilgilerini koy
	info := SystemInfo{
		TotalRAM:    float32(v.Total) / 1024.0 / 1024.0 / 1024.0,
		UsedRAM:     float32(v.Used) / 1024.0 / 1024.0 / 1024.0,
		UsedPercent: float32(v.UsedPercent),

		// swap verilerini de byte'dan GB'a çeviriyoruz
		TotalSwap:   float32(s.Total) / 1024.0 / 1024.0 / 1024.0,
		UsedSwap:    float32(s.Used) / 1024.0 / 1024.0 / 1024.0,
		SwapPercent: float32(s.UsedPercent),

		CPUPercent:  cpuKullanimi,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func main() {
	http.HandleFunc("/api/status", statusHandler)

	fmt.Println("Sunucu 8080 portunda calismaya basladi. http://localhost:8080/api/status adresine gidebilirsin.")

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("HATA:", err)
	}
}
