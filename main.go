package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/distatus/battery"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	psnet "github.com/shirou/gopsutil/v3/net" //"net" paketiyle aynı ada sahip olduğundan çakışmasını önlemek için psnet dedik

	"net" // IP adresi için
	"net/http"
	"os"
	"os/signal" //graceful shutdown için
	"syscall"
	"time"
)

// json şablonumuz(blueprint gibi) (struct)
type SystemInfo struct {
	OS       string `json:"isletim_sistemi"` // bellekte string tutacak OS adında bi alan (değişken) aç. API'den json olarak yollarken adını "isletim_sistemi" diye değiştir, çünkü frontend tarafı genelde küçük harfli isimlendirme bekliyor.
	Kernel   string `json:"kernel_surumu"`
	Hostname string `json:"bilgisayar_adi"`
	Uptime   uint64 `json:"calisma_suresi_sn"`
	LocalIP  string `json:"yerel_ip"`

	CPUModel   string  `json:"cpu_modeli"`
	CPUPercent float32 `json:"cpu_kullanim_orani"`
	CPUTemp    float32 `json:"cpu_sicaklik_derece"`

	RAMPercent  float32 `json:"ram_kullanim_orani"`
	RAMUsedByte uint64  `json:"ram_kullanilan_byte"`

	DiskTotalByte uint64  `json:"disk_toplam_byte"`
	DiskUsedByte  uint64  `json:"disk_kullanilan_byte"`
	DiskPercent   float32 `json:"disk_kullanim_orani"`

	BatteryPrcnt float32 `json:"batarya_yuzdesi"`
	IsCharging   bool    `json:"sarj_oluyor_mu"`

	NetSentByte uint64 `json:"ag_gonderilen_byte"`
	NetRecvByte uint64 `json:"ag_alinan_byte"`
}

// cihaz IP bulma kodu
func getLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs() //gelen hata err atanır, cevap ise addrs atanır.
	if err != nil {
		return "", err //1. değişken(string) boş döndür, 2. değişken(error) err döndür (string, error)
	}

	for _, address := range addrs { //verilen indexi blank (_) atar, ipleri address e atar
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil //1. değişken(string) ip döndür, err olmadığı için null döndür error değişkeni yerine
			}
		}
	}

	return "IP bulunamadi", nil //string'i ip bulunamadı döndür, err null olsun
}

func statusHandler(w http.ResponseWriter, r *http.Request) {

	//sadece GET metodu izin veriyoruz
	if r.Method != http.MethodGet {
		http.Error(w, "Only method GET is allowed", http.StatusMethodNotAllowed)
		return
	}
	// CORS izinleri, frontend etkileşime geçebilsin diye
	w.Header().Set("Access-Control-Allow-Origin", "*")             //local olarak çalışacağından herkes girebilsin yapıyoruz
	w.Header().Set("Access-Control-Allow-Methods", "GET")          //sadece GET metoduna izin ver
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type") // allows Content-Type header for CORS/JSON requests

	// host (sistem) bilgileri
	hInfo, err := host.Info()
	if err != nil {
		http.Error(w, "Host bilgisi okunamadi", http.StatusInternalServerError)
		return
	}

	// ram bilgileri
	v, err := mem.VirtualMemory()
	if err != nil {
		http.Error(w, "RAM bilgisi okunamadi", http.StatusInternalServerError)
		return
	}

	// CPU kullanım bilgileri
	// ilk parametre ne kadar sürelik bir ölçüm yapacağı
	// ikinci parametre (false) çekirdek ortalamasını verir
	cPercent, err := cpu.Percent(100*time.Millisecond, false)
	if err != nil {
		http.Error(w, "CPU kullanim bilgisi okunamadi", http.StatusInternalServerError)
		return
	}
	// işlemci modeli bilgisi
	cInfo, err := cpu.Info()
	if err != nil {
		http.Error(w, "CPU model bilgisi okunamadi", http.StatusInternalServerError)
		return
	}
	//cpu verisi okunamazsa hata ver
	if len(cPercent) == 0 || len(cInfo) == 0 {
		http.Error(w, "CPU verisi alinamadi", http.StatusInternalServerError)
		return
	}

	// disk bilgileri (ana dizin)
	dInfo, err := disk.Usage("/")
	if err != nil {
		http.Error(w, "Disk bilgisi okunamadi", http.StatusInternalServerError)
		return
	}

	//ip bulma fonksiyonundan ip veya error alınır
	localIP, err := getLocalIP()
	if err != nil {
		http.Error(w, "Yerel IP bilgisi okunamadi", http.StatusInternalServerError)
		return
	}

	// ağ trafiği bilgileri (toplam upload/download byte)
	netStats, err := psnet.IOCounters(false)
	if err != nil {
		http.Error(w, "Ag bilgisi okunamadi", http.StatusInternalServerError)
		return
	}
	//ağ verisi alınamazsa hata ver
	if len(netStats) == 0 {
		http.Error(w, "Ag verisi alinamadi", http.StatusInternalServerError)
		return
	}

	// sıcaklık sensörlerini okuyup en yüksek sıcaklığı alıyoruz
	tempStats, err := host.SensorsTemperatures()
	if err != nil {
		http.Error(w, "Sicaklik bilgisi okunamadi", http.StatusInternalServerError)
		return
	}
	//en yüksek dereceyi alıp onu gösteriyoruz
	maxTemp := 0.0
	for _, temp := range tempStats {
		if temp.Temperature > maxTemp {
			maxTemp = temp.Temperature
		}
	}

	// batarya bilgisi laptop olmayan cihazlarda olmayabilir, o yüzden esnek gidiyoruz
	bats, err := battery.GetAll()
	batPercent := 0.0
	isCharging := false
	if err == nil && len(bats) > 0 && bats[0].Full > 0 {
		batPercent = (bats[0].Current / bats[0].Full) * 100
		isCharging = (bats[0].State.String() == "Charging")
	}

	// şablon(blueprint) (struct) içini sistem bilgileriyle doldur
	info := SystemInfo{
		OS:          hInfo.OS, //şablondaki OS kısmına hInfo.OS içindeki değeri(stringi) yaz, yani şablona verileri doldurmaya başla
		Kernel:      hInfo.KernelVersion,
		Hostname:    hInfo.Hostname,
		Uptime:      hInfo.Uptime,
		LocalIP:     localIP,
		CPUModel:    cInfo[0].ModelName,
		CPUPercent:  float32(cPercent[0]),
		CPUTemp:     float32(maxTemp),
		RAMPercent:  float32(v.UsedPercent),
		DiskPercent: float32(dInfo.UsedPercent),

		BatteryPrcnt: float32(batPercent),
		IsCharging:   isCharging,

		//toplam indirme/yükleme. anlık indirme/yükleme hızını frontend de gösterirken hesaplayacağız
		NetSentByte: netStats[0].BytesSent,
		NetRecvByte: netStats[0].BytesRecv,

		//değerleri byte olarak yolluyoruz, frontend de gösterirken megabyte veya gigabyte'a çevireceğiz
		RAMUsedByte:   v.Used,
		DiskTotalByte: dInfo.Total,
		DiskUsedByte:  dInfo.Used,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(info); err != nil {
		fmt.Printf("JSON encode hatasi: %v\n", err) //JSON yaparken sorun çıkarsa hatayı logluyoruz
	}
}

func main() {
	//http server başlatılıyor
	http.HandleFunc("/api/status", statusHandler)

	srv := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	go func() {
		fmt.Println("Sunucu 8080 portunda calismaya basladi. http://localhost:8080/api/status adresinde.")

		//hata yakalayıcı
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			fmt.Println("HATA:", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	fmt.Println("\nKapatma sinyali alindi. Sunucu kapatiliyor...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := srv.Shutdown(ctx)
	if err != nil {
		fmt.Println("HATA:", err)
	}
}
