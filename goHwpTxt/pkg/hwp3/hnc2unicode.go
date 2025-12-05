package hwp3

import (
	"strings"
)

func HncToUtf8(c uint16) string {
	var sb strings.Builder

	if c >= 0x0020 && c <= 0x007e {
		sb.WriteRune(rune(c))
		return sb.String()
	} else if c >= 0x007f && c <= 0x3fff {
		return _hnc_to_utf8(c)
	} else if c >= 0x4000 && c <= 0x5317 {
		// 1수준 한자
		idx := int(c) - 0x4000
		if idx >= 0 && idx < len(ksc5601_2uni_page4a) {
			sb.WriteRune(rune(ksc5601_2uni_page4a[idx]))
			return sb.String()
		}
		return ""
	} else if c >= 0x5318 && c <= 0x7fff {
		return _hnc_to_utf8(c)
	} else if c >= 0x8000 {
		l := (c & 0x7c00) >> 10 // 초성
		v := (c & 0x03e0) >> 5  // 중성
		t := (c & 0x001f)       // 종성

		if int(l) >= len(L_MAP) || int(v) >= len(V_MAP) || int(t) >= len(T_MAP) {
			return ""
		}

		if L_MAP[l] != NONE && V_MAP[v] != NONE && T_MAP[t] != NONE {
			// Modern Hangul Syllable
			syllable := 0xac00 + (uint32(L_MAP[l]) * 21 * 28) + (uint32(V_MAP[v]) * 28) + uint32(T_MAP[t])
			sb.WriteRune(rune(syllable))
			return sb.String()
		} else if HNC_L1[v] != FILL && (HNC_V1[v] == FILL || HNC_V1[v] == NONE) && HNC_T1[t] == FILL {
			if HNC_L1[l] != FILL && (HNC_V1[v] == FILL || HNC_V1[v] == NONE) && HNC_T1[t] == FILL {
				sb.WriteRune(rune(HNC_L1[l]))
				return sb.String()
			} else if HNC_L1[l] == FILL && (HNC_V1[v] != FILL && HNC_V1[v] != NONE) && HNC_T1[t] == FILL {
				sb.WriteRune(rune(HNC_V1[v]))
				return sb.String()
			} else if HNC_L1[l] == FILL && (HNC_V1[v] == FILL || HNC_V1[v] == NONE) && HNC_T1[t] != FILL {
				sb.WriteRune(rune(HNC_T1[t]))
				return sb.String()
			} else if HNC_L1[l] != FILL && (HNC_V1[v] != FILL && HNC_V1[v] != NONE) && HNC_T1[t] == FILL {
				// Old Hangul L+V
				sb.WriteRune(rune(HNC_L2[l]))
				sb.WriteRune(rune(HNC_V2[v]))
				return sb.String()
			} else if HNC_L1[l] != FILL && (HNC_V1[v] != FILL && HNC_V1[v] != NONE) && HNC_T1[t] != FILL {
				// Old Hangul L+V+T
				sb.WriteRune(rune(HNC_L2[l]))
				sb.WriteRune(rune(HNC_V2[v]))
				sb.WriteRune(rune(HNC_T2[t]))
				return sb.String()
			} else if v == 0 {
				// Completed Old Hangul?
				res := _hnc_to_utf8(c)
				if res != "" {
					return res
				}
			}
		}
	}
	return ""
}

func _hnc_to_utf8(c uint16) string {
	var sb strings.Builder
	switch c {
	case 0xbc1f: // 르ᇝ
		sb.WriteRune(0x1105)
		sb.WriteRune(0x1173)
		sb.WriteRune(0x11dd)
	case 0xd802: // 아ᇇ
		sb.WriteRune(0x110b)
		sb.WriteRune(0x1161)
		sb.WriteRune(0x11c7)
	default:
		if val, ok := hnc2uni_map[c]; ok {
			sb.WriteRune(rune(val))
		} else {
			return ""
		}
	}
	return sb.String()
}
