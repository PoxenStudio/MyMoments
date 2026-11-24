export interface CityOption {
  name: string;
  lat: number;
  lon: number;
}

export interface ProvinceOption {
  name: string;
  cities: CityOption[];
}

// 离线省市数据，覆盖省级行政区与主要城市，用于手动选择天气位置。
export const provinces: ProvinceOption[] = [
  { name: '北京', cities: [{ name: '北京', lat: 39.9042, lon: 116.4074 }] },
  { name: '天津', cities: [{ name: '天津', lat: 39.0842, lon: 117.2009 }] },
  { name: '河北', cities: [{ name: '石家庄', lat: 38.0428, lon: 114.5149 }, { name: '唐山', lat: 39.6305, lon: 118.1802 }, { name: '保定', lat: 38.874, lon: 115.4646 }] },
  { name: '山西', cities: [{ name: '太原', lat: 37.8706, lon: 112.5489 }, { name: '大同', lat: 40.0768, lon: 113.3001 }] },
  { name: '内蒙古', cities: [{ name: '呼和浩特', lat: 40.8426, lon: 111.7492 }, { name: '包头', lat: 40.6574, lon: 109.8403 }] },
  { name: '辽宁', cities: [{ name: '沈阳', lat: 41.8057, lon: 123.4315 }, { name: '大连', lat: 38.914, lon: 121.6147 }] },
  { name: '吉林', cities: [{ name: '长春', lat: 43.8171, lon: 125.3235 }, { name: '吉林', lat: 43.8378, lon: 126.5495 }] },
  { name: '黑龙江', cities: [{ name: '哈尔滨', lat: 45.8038, lon: 126.5349 }, { name: '齐齐哈尔', lat: 47.3543, lon: 123.918 }] },
  { name: '上海', cities: [{ name: '上海', lat: 31.2304, lon: 121.4737 }] },
  { name: '江苏', cities: [{ name: '南京', lat: 32.0603, lon: 118.7969 }, { name: '苏州', lat: 31.2989, lon: 120.5853 }, { name: '无锡', lat: 31.4912, lon: 120.3119 }] },
  { name: '浙江', cities: [{ name: '杭州', lat: 30.2741, lon: 120.1551 }, { name: '宁波', lat: 29.8683, lon: 121.544 }, { name: '温州', lat: 27.994, lon: 120.6994 }] },
  { name: '安徽', cities: [{ name: '合肥', lat: 31.8206, lon: 117.2272 }, { name: '芜湖', lat: 31.3525, lon: 118.4331 }] },
  { name: '福建', cities: [{ name: '福州', lat: 26.0745, lon: 119.2965 }, { name: '厦门', lat: 24.4798, lon: 118.0894 }, { name: '泉州', lat: 24.8741, lon: 118.6757 }] },
  { name: '江西', cities: [{ name: '南昌', lat: 28.682, lon: 115.8579 }, { name: '赣州', lat: 25.8309, lon: 114.935 }] },
  { name: '山东', cities: [{ name: '济南', lat: 36.6512, lon: 117.1201 }, { name: '青岛', lat: 36.0671, lon: 120.3826 }, { name: '烟台', lat: 37.4638, lon: 121.4479 }] },
  { name: '河南', cities: [{ name: '郑州', lat: 34.7466, lon: 113.6254 }, { name: '洛阳', lat: 34.6197, lon: 112.454 }] },
  { name: '湖北', cities: [{ name: '武汉', lat: 30.5928, lon: 114.3055 }, { name: '宜昌', lat: 30.6919, lon: 111.2865 }] },
  { name: '湖南', cities: [{ name: '长沙', lat: 28.2282, lon: 112.9388 }, { name: '株洲', lat: 27.8274, lon: 113.134 }] },
  { name: '广东', cities: [{ name: '广州', lat: 23.1291, lon: 113.2644 }, { name: '深圳', lat: 22.5431, lon: 114.0579 }, { name: '珠海', lat: 22.2707, lon: 113.5767 }, { name: '佛山', lat: 23.0215, lon: 113.1214 }] },
  { name: '广西', cities: [{ name: '南宁', lat: 22.817, lon: 108.3665 }, { name: '桂林', lat: 25.2736, lon: 110.2902 }] },
  { name: '海南', cities: [{ name: '海口', lat: 20.0442, lon: 110.1999 }, { name: '三亚', lat: 18.2528, lon: 109.5119 }] },
  { name: '重庆', cities: [{ name: '重庆', lat: 29.563, lon: 106.5516 }] },
  { name: '四川', cities: [{ name: '成都', lat: 30.5728, lon: 104.0668 }, { name: '绵阳', lat: 31.4675, lon: 104.6796 }] },
  { name: '贵州', cities: [{ name: '贵阳', lat: 26.647, lon: 106.6302 }, { name: '遵义', lat: 27.7257, lon: 106.9272 }] },
  { name: '云南', cities: [{ name: '昆明', lat: 25.0389, lon: 102.7183 }, { name: '大理', lat: 25.6065, lon: 100.2676 }, { name: '丽江', lat: 26.8721, lon: 100.2299 }] },
  { name: '西藏', cities: [{ name: '拉萨', lat: 29.652, lon: 91.1721 }] },
  { name: '陕西', cities: [{ name: '西安', lat: 34.3416, lon: 108.9398 }, { name: '宝鸡', lat: 34.3619, lon: 107.2373 }] },
  { name: '甘肃', cities: [{ name: '兰州', lat: 36.0611, lon: 103.8343 }, { name: '天水', lat: 34.5809, lon: 105.7249 }] },
  { name: '青海', cities: [{ name: '西宁', lat: 36.6171, lon: 101.7782 }] },
  { name: '宁夏', cities: [{ name: '银川', lat: 38.4872, lon: 106.2309 }] },
  { name: '新疆', cities: [{ name: '乌鲁木齐', lat: 43.8256, lon: 87.6168 }, { name: '喀什', lat: 39.4704, lon: 75.9898 }] },
  { name: '香港', cities: [{ name: '香港', lat: 22.3193, lon: 114.1694 }] },
  { name: '澳门', cities: [{ name: '澳门', lat: 22.1987, lon: 113.5439 }] },
  { name: '台湾', cities: [{ name: '台北', lat: 25.033, lon: 121.5654 }, { name: '高雄', lat: 22.6273, lon: 120.3014 }] },
];

export function nearestCity(lat: number, lon: number): CityOption | null {
  let best: CityOption | null = null;
  let bestDistance = Number.POSITIVE_INFINITY;
  for (const province of provinces) {
    for (const city of province.cities) {
      const dLat = city.lat - lat;
      const dLon = city.lon - lon;
      const distance = dLat * dLat + dLon * dLon;
      if (distance < bestDistance) {
        bestDistance = distance;
        best = city;
      }
    }
  }
  return best;
}
