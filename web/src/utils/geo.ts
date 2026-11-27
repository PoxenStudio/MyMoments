import { api } from '@/api/client';
import { findCityByName, nearestCity, type CityOption } from '@/data/cities';

// 浏览器直连出口 IP 查询服务；受 CORS / CSP 拦截时自动回退到服务端代理。
const BROWSER_LOOKUPS: Array<() => Promise<CityOption | null>> = [lookupIPIP, lookupIPWho];
const LOOKUP_TIMEOUT = 3500;

interface GeoIPResponse {
  city?: string;
  lat?: number;
  lon?: number;
}

async function fetchText(url: string): Promise<string> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), LOOKUP_TIMEOUT);
  try {
    const res = await fetch(url, {
      signal: controller.signal,
      credentials: 'omit',
      cache: 'no-store',
      referrerPolicy: 'no-referrer',
    });
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    return await res.text();
  } finally {
    clearTimeout(timer);
  }
}

// 解析 "当前 IP：1.2.3.4  来自于：中国 广东 深圳  电信"。
async function lookupIPIP(): Promise<CityOption | null> {
  const text = await fetchText('https://myip.ipip.net/');
  const match = /来自于[：:]\s*(.+)/.exec(text);
  if (!match) return null;
  const parts = match[1].trim().split(/\s+/);
  const city = parts.length >= 3 ? parts[2] : parts[1];
  return city ? findCityByName(city) : null;
}

// ipwho.is 的城市多为英文，用坐标在离线城市表中就近匹配。
async function lookupIPWho(): Promise<CityOption | null> {
  const text = await fetchText('https://ipwho.is/');
  const data = JSON.parse(text) as { success?: boolean; latitude?: number; longitude?: number };
  if (data.success === false || typeof data.latitude !== 'number' || typeof data.longitude !== 'number') {
    return null;
  }
  return nearestCity(data.latitude, data.longitude);
}

// 先让浏览器直连上游查询出口 IP，全部失败时改由服务端请求，避开 CORS 与 HTTP 页面限制。
export async function detectCityByIP(): Promise<CityOption | null> {
  for (const lookup of BROWSER_LOOKUPS) {
    try {
      const city = await lookup();
      if (city) return city;
    } catch {
      // 继续尝试下一个地址
    }
  }
  try {
    const data = await api.get<GeoIPResponse>('/geo/ip');
    if (typeof data.lat === 'number' && typeof data.lon === 'number') return nearestCity(data.lat, data.lon);
    return data.city ? findCityByName(data.city) : null;
  } catch {
    return null;
  }
}
