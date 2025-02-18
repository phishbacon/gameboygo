package cart

var NewLicCodes = map[uint16]string{
	12336: "None",
	12337: "Nintendo Research & Development 1",
	12344: "Capcom",
	12595: "EA (Electronic Arts)",
	12600: "Hudson Soft",
	12601: "B-AI",
	12848: "KSS",
	12850: "Planning Office WADA",
	12852: "PCM Complete",
	12853: "San-X",
	12856: "Kemco",
	12857: "SETA Corporation",
	13104: "Viacom",
	13105: "Nintendo",
	13106: "Bandai",
	13107: "Ocean Software/Acclaim Entertainment",
	13108: "Konami",
	13109: "HectorSoft",
	13111: "Taito",
	13112: "Hudson Soft",
	13113: "Banpresto",
	13361: "Ubi Soft1",
	13362: "Atlus",
	13364: "Malibu Interactive",
	13366: "Angel",
	13367: "Bullet-Proof Software2",
	13369: "Irem",
	13616: "Absolute",
	13617: "Acclaim Entertainment",
	13618: "Activision",
	13619: "Sammy USA Corporation",
	13620: "Konami",
	13621: "Hi Tech Expressions",
	13622: "LJN",
	13623: "Matchbox",
	13624: "Mattel",
	13625: "Milton Bradley Company",
	13872: "Titus Interactive",
	13873: "Virgin Games Ltd.3",
	13876: "Lucasfilm Games4",
	13879: "Ocean Software",
	13881: "EA (Electronic Arts)",
	14128: "Infogrames5",
	14129: "Interplay Entertainment",
	14130: "Broderbund",
	14131: "Sculptured Software6",
	14133: "The Sales Curve Limited7",
	14136: "THQ",
	14137: "Accolade",
	14384: "Misawa Entertainment",
	14387: "lozc",
	14390: "Tokuma Shoten",
	14391: "Tsukuda Original",
	14641: "Chunsoft Co.8",
	14642: "Video System",
	14643: "Ocean Software/Acclaim Entertainment",
	14645: "Varie",
	14646: "Yonezawa/s’pal",
	14647: "Kaneko",
	14649: "Pack-In-Video",
	14664: "Bottom Up",
	16692: "Konami (Yu-Gi-Oh!)",
	16972: "MTO",
	17483: "Kodansha",
}

var Types = map[uint8]string{
	0x00: "ROM ONLY",
	0x01: "MBC1",
	0x02: "MBC1+RAM",
	0x03: "MBC1+RAM+BATTERY",
	0x05: "MBC2",
	0x06: "MBC2+BATTERY",
	0x0B: "MMM01",
	0x0C: "MMM01+RAM",
	0x0D: "MMM01+RAM+BATTERY",
	0x0F: "MBC3+TIMER+BATTERY",
	0x10: "MBC3+TIMER+RAM+BATTERY 10",
	0x11: "MBC3",
	0x12: "MBC3+RAM 10",
	0x13: "MBC3+RAM+BATTERY 10",
	0x19: "MBC5",
	0x1A: "MBC5+RAM",
	0x1B: "MBC5+RAM+BATTERY",
	0x1C: "MBC5+RUMBLE",
	0x1D: "MBC5+RUMBLE+RAM",
	0x1E: "MBC5+RUMBLE+RAM+BATTERY",
	0x20: "MBC6",
	0x22: "MBC7+SENSOR+RUMBLE+RAM+BATTERY",
}

type Cart struct {
	Entry          [0x0004]uint8
	Logo           [0x0030]uint8
	Title          [0x0010]byte
	NewLicCode     uint16
	SGBFlag        uint8
	Type           uint8
	ROMSize        uint8
	RAMSize        uint8
	DestCode       uint8
	OldLicCode     uint8
	Version        uint8
	HeaderChecksum uint8
	GlobalChecksum uint8
}

func (c Cart) GetCartLicName() string {
	if code, ok := NewLicCodes[c.NewLicCode]; ok {
		return code
	} else {
		return "UNKOWN LICENSE CODE"
	}
}

func (c Cart) GetCartTypeName() string {
  if t, ok := Types[c.Type]; ok {
    return t
  } else {
    return "UNKNOWN TYPE"
  }
}
