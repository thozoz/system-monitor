"use client";

import { useEffect, useState } from "react";

type SystemInfo = {
  hostname: string;
  local_ip: string;
  operating_system: string;
  kernel_version: string;
  uptime_seconds: number;
  cpu_model: string;
  cpu_usage_percent: number;
  cpu_temperature_celsius: number;
  ram_usage_percent: number;
  ram_used_bytes: number;
  disk_total_bytes: number;
  disk_used_bytes: number;
  disk_usage_percent: number;
  battery_percent: number;
  is_charging: boolean;
  network_sent_bytes: number;
  network_received_bytes: number;
};

function formatBytes(bytes: number): string {
  if (bytes <= 0) return "0 B";
  if (bytes >= 1073741824) return (bytes / 1073741824).toFixed(1) + " GB";
  if (bytes >= 1048576) return (bytes / 1048576).toFixed(1) + " MB";
  return (bytes / 1024).toFixed(1) + " KB";
}

function formatUptime(seconds: number): string {
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  return `${h}h ${m}m`;
}

function ProgressBar({ value, color = "bg-zinc-400" }: { value: number; color?: string }) {
  const clamped = Math.min(Math.max(value, 0), 100);
  const barColor = value > 85 ? "bg-red-500" : value > 60 ? "bg-yellow-500" : color;
  return (
    <div className="h-1.5 w-full rounded-full bg-zinc-800">
      <div
        className={`h-1.5 rounded-full transition-all duration-500 ${barColor}`}
        style={{ width: `${clamped}%` }}
      />
    </div>
  );
}

function StatCard({
  label,
  value,
  sub,
  percent,
}: {
  label: string;
  value: string;
  sub?: string;
  percent?: number;
}) {
  return (
    <div className="border border-zinc-800 bg-zinc-950 p-4 rounded">
      <p className="text-xs text-zinc-500 uppercase tracking-widest mb-2">{label}</p>
      <p className="text-xl font-semibold text-white">{value}</p>
      {sub && <p className="text-xs text-zinc-500 mt-1">{sub}</p>}
      {percent !== undefined && percent >= 0 && (
        <div className="mt-3">
          <ProgressBar value={percent} />
          <p className="text-xs text-zinc-600 mt-1">{percent.toFixed(1)}%</p>
        </div>
      )}
    </div>
  );
}

export default function Dashboard() {
  const [sysInfo, setSysInfo] = useState<SystemInfo | null>(null);
  const [error, setError] = useState(false);
  const [netSpeed, setNetSpeed] = useState<{ up: number; down: number }>({ up: 0, down: 0 });

  useEffect(() => {
    const apiPort = process.env.NEXT_PUBLIC_API_PORT || "8080";
    const fetchData = () => {
      fetch(`http://localhost:${apiPort}/api/status`)
        .then((res) => res.json())
        .then((data: SystemInfo) => {
          setError(false);
          setSysInfo((prev) => {
            if (prev) {
              setNetSpeed({
                up: (data.network_sent_bytes - prev.network_sent_bytes) / 2,
                down: (data.network_received_bytes - prev.network_received_bytes) / 2,
              });
            }
            return data;
          });
        })
        .catch(() => setError(true));
    };

    fetchData();
    const interval = setInterval(fetchData, 2000);
    return () => clearInterval(interval);
  }, []);

  if (error) {
    const apiPort = process.env.NEXT_PUBLIC_API_PORT || "8080";
    return (
      <div className="flex min-h-screen items-center justify-center bg-black font-mono text-red-500">
        Cannot connect to Go server at localhost:{apiPort}
      </div>
    );
  }

  if (!sysInfo) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-black font-mono text-zinc-600">
        Waiting for data...
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-black px-8 py-10 font-mono text-gray-300">
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-2xl font-bold text-white tracking-tight">
          {sysInfo.hostname}
        </h1>
        <p className="text-sm text-zinc-500 mt-1">
          {sysInfo.operating_system} · {sysInfo.kernel_version} · {sysInfo.local_ip} · up {formatUptime(sysInfo.uptime_seconds)}
        </p>
      </div>

      {/* CPU */}
      <section className="mb-6">
        <p className="text-xs text-zinc-600 uppercase tracking-widest mb-3">Processor</p>
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
          <div className="col-span-2 border border-zinc-800 bg-zinc-950 p-4 rounded">
            <p className="text-xs text-zinc-500 uppercase tracking-widest mb-2">Model</p>
            <p className="text-sm text-white">{sysInfo.cpu_model}</p>
          </div>
          <StatCard
            label="Usage"
            value={`${sysInfo.cpu_usage_percent.toFixed(1)}%`}
            percent={sysInfo.cpu_usage_percent}
          />
        </div>
        {sysInfo.cpu_temperature_celsius >= 0 && (
          <div className="mt-3 border border-zinc-800 bg-zinc-950 p-4 rounded">
            <p className="text-xs text-zinc-500 uppercase tracking-widest mb-1">Temperature</p>
            <p className="text-lg font-semibold text-white">{sysInfo.cpu_temperature_celsius.toFixed(0)}°C</p>
          </div>
        )}
      </section>

      {/* RAM + Disk */}
      <section className="mb-6">
        <p className="text-xs text-zinc-600 uppercase tracking-widest mb-3">Memory & Storage</p>
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <StatCard
            label="RAM"
            value={formatBytes(sysInfo.ram_used_bytes)}
            sub="used"
            percent={sysInfo.ram_usage_percent}
          />
          {sysInfo.disk_usage_percent >= 0 ? (
            <StatCard
              label="Disk"
              value={formatBytes(sysInfo.disk_used_bytes)}
              sub={`of ${formatBytes(sysInfo.disk_total_bytes)}`}
              percent={sysInfo.disk_usage_percent}
            />
          ) : (
            <StatCard label="Disk" value="N/A" />
          )}
        </div>
      </section>

      {/* Battery + Network */}
      <section>
        <p className="text-xs text-zinc-600 uppercase tracking-widest mb-3">Battery & Network</p>
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          {sysInfo.battery_percent >= 0 ? (
            <StatCard
              label="Battery"
              value={`${sysInfo.battery_percent.toFixed(0)}%`}
              sub={sysInfo.is_charging ? "charging" : "discharging"}
              percent={sysInfo.battery_percent}
            />
          ) : (
            <StatCard label="Battery" value="N/A" sub="not available" />
          )}
          <div className="border border-zinc-800 bg-zinc-950 p-4 rounded">
            <p className="text-xs text-zinc-500 uppercase tracking-widest mb-2">Network</p>
            <div className="space-y-1">
              <p className="text-sm text-white">↑ {formatBytes(netSpeed.up)}/s</p>
              <p className="text-sm text-white">↓ {formatBytes(netSpeed.down)}/s</p>
              <p className="text-xs text-zinc-600 mt-2">
                Total: ↑{formatBytes(sysInfo.network_sent_bytes)} ↓{formatBytes(sysInfo.network_received_bytes)}
              </p>
            </div>
          </div>
        </div>
      </section>
    </div>
  );
}
