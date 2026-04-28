"use client"; // Backend değil, saf tarayıcı kodu olduğumuzu belirttik

import { useEffect, useState } from "react";

type SystemInfo = {
  //struct yapıldı
  hostname: string;
  local_ip: string;
  operating_system: string;
  kernel_version: string;
  cpu_model: string;
  cpu_usage_percent: number;
  cpu_temperature_celsius: number;
  ram_usage_percent: number;
};

export default function Dashboard() {
  // 1. HAFIZA: sysInfo adında bir değişken açtık, başlangıç değeri null (boş).
  // setSysInfo ise bu değişkeni değiştirmemize yarayan özel kumandamız.
  const [sysInfo, setSysInfo] = useState<SystemInfo | null>(null);

  // 2. KONTROL: Saniyede bir Go'ya istek atan motorumuz
  useEffect(() => {
    // Istek atma isini bir fonksiyona bagladik ki tekrar tekrar cagirabilelim
    const verileriCek = () => {
      fetch("http://localhost:8080/api/status")
        .then((cevap) => cevap.json())
        .then((veri) => {
          setSysInfo(veri); // Gelen taze veriyi hafizaya yaz (Ekranda sayilar aninda degisecek!)
        })
        .catch((hata) => console.log("Go sunucusuna ulasilamadi:", hata));
    };

    // Sayfa ilk acildiginda 1 kere cek (Beklememek icin)
    verileriCek();

    // ZAMANLAYICI (Interval): Her 1000 milisaniyede (1 saniye) bir 'verileriCek' fonksiyonunu tetikle
    const motor = setInterval(verileriCek, 1000);

    // TEMIZLIK (Cleanup): React sayfasi kapatilirsa arka planda calisan zamanlayiciyi yok et.
    // (Go'daki graceful shutdown mantiginin frontend versiyonu, RAM sismesini onler)
    return () => clearInterval(motor);
  }, []);

  // 1. GÜVENLİK DUVARI: sysInfo henüz boşsa (veri yoldaysa) burası çalışır.
  if (!sysInfo) {
    return (
      <div className="flex min-h-screen items-start bg-black px-12 py-12 font-mono text-green-500">
        Go Sunucusundan Veri Bekleniyor...
      </div>
    );
  }

  // 2. ANA EKRAN: sysInfo dolduğunda (veri geldiğinde) React otomatik olarak burayı çizer.
  return (
    <div className="min-h-screen bg-black px-12 py-12 font-mono text-gray-300">
      <h1 className="text-3xl font-bold text-white">FEDORA SİSTEM MONİTÖRÜ</h1>
      <hr className="mt-4 mb-6 border-zinc-800" />

      <div className="space-y-1">
        <p>
          <b>Bilgisayar:</b> {sysInfo.hostname} ({sysInfo.local_ip})
        </p>
        <p>
          <b>İşletim Sistemi:</b> {sysInfo.operating_system} -{" "}
          {sysInfo.kernel_version}
        </p>
      </div>

      <div className="mt-6 space-y-1">
        <p>
          <b>İşlemci:</b> {sysInfo.cpu_model}
        </p>
        <p>
          <b>CPU Kullanımı:</b> %{sysInfo.cpu_usage_percent.toFixed(1)}
        </p>
        <p>
          <b>CPU Sıcaklığı:</b> {sysInfo.cpu_temperature_celsius}°C
        </p>
      </div>

      <div className="mt-6 space-y-1">
        <p>
          <b>RAM Kullanımı:</b> %{sysInfo.ram_usage_percent.toFixed(1)}
        </p>
      </div>
    </div>
  );
}
