// Copyright  observIQ, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package goflow

func protoName(x int) (name string) {
	switch x {
	case 0:
		name = "HOPOPT"
	case 1:
		name = "ICMP"
	case 2:
		name = "IGMP"
	case 3:
		name = "GGP"
	case 4:
		name = "IP-in-IP"
	case 5:
		name = "ST"
	case 6:
		name = "TCP"
	case 7:
		name = "CBT"
	case 8:
		name = "EGP"
	case 9:
		name = "IGP"
	case 10:
		name = "BBN-RCC-MON"
	case 11:
		name = "NVP-II"
	case 12:
		name = "PUP"
	case 13:
		name = "ARGUS"
	case 14:
		name = "EMCON"
	case 15:
		name = "XNET"
	case 16:
		name = "CHAOS"
	case 17:
		name = "UDP"
	case 18:
		name = "MUX"
	case 19:
		name = "DCN-MEAS"
	case 20:
		name = "HMP"
	case 21:
		name = "PRM"
	case 22:
		name = "XNS-IDP"
	case 23:
		name = "TRUNK-1"
	case 24:
		name = "TRUNK-2"
	case 25:
		name = "LEAF-1"
	case 26:
		name = "LEAF-2"
	case 27:
		name = "RDP"
	case 28:
		name = "IRTP"
	case 29:
		name = "ISO-TP4"
	case 30:
		name = "NETBLT"
	case 31:
		name = "MFE-NSP"
	case 32:
		name = "MERIT-INP"
	case 33:
		name = "DCCP"
	case 34:
		name = "3PC"
	case 35:
		name = "IDPR"
	case 36:
		name = "XTP"
	case 37:
		name = "DDP"
	case 38:
		name = "IDPR-CMTP"
	case 39:
		name = "TP++"
	case 40:
		name = "IL"
	case 41:
		name = "IPv6"
	case 42:
		name = "SDRP"
	case 43:
		name = "IPv6-Route"
	case 44:
		name = "IPv6-Frag"
	case 45:
		name = "IDRP"
	case 46:
		name = "RSVP"
	case 47:
		name = "GRE"
	case 48:
		name = "DSR"
	case 49:
		name = "BNA"
	case 50:
		name = "ESP"
	case 51:
		name = "AH"
	case 52:
		name = "I-NLSP"
	case 53:
		name = "SwIPe"
	case 54:
		name = "NARP"
	case 55:
		name = "MOBILE"
	case 56:
		name = "TLSP"
	case 57:
		name = "SKIP"
	case 58:
		name = "IPv6-ICMP"
	case 59:
		name = "IPv6-NoNxt"
	case 60:
		name = "IPv6-Opts"
	case 61:
		name = "Host Internal Protocol"
	case 62:
		name = "CFTP"
	case 63:
		name = "Local Network"
	case 64:
		name = "SAT-EXPAK"
	case 65:
		name = "KRYPTOLAN"
	case 66:
		name = "RVD"
	case 67:
		name = "IPPC"
	case 68:
		name = "Distribution File System"
	case 69:
		name = "SAT-MON"
	case 70:
		name = "VISA"
	case 71:
		name = "IPCU"
	case 72:
		name = "CPNX"
	case 73:
		name = "CPHB"
	case 74:
		name = "WSN"
	case 75:
		name = "PVP"
	case 76:
		name = "BR-SAT-MON"
	case 77:
		name = "SUN-ND"
	case 78:
		name = "WB-MON"
	case 79:
		name = "WB-EXPAK"
	case 80:
		name = "ISO-IP"
	case 81:
		name = "VMTP"
	case 82:
		name = "SECURE-VMTP"
	case 83:
		name = "VINES"
	case 84:
		name = "TTP/IPTM"
	case 85:
		name = "NSFNET-IGP"
	case 86:
		name = "DGP"
	case 87:
		name = "TCF"
	case 88:
		name = "EIGRP"
	case 89:
		name = "OSPF"
	case 90:
		name = "Sprite-RPC"
	case 91:
		name = "LARP"
	case 92:
		name = "MTP"
	case 93:
		name = "AX.25"
	case 94:
		name = "OS"
	case 95:
		name = "MICP"
	case 96:
		name = "SCC-SP"
	case 97:
		name = "ETHERIP"
	case 98:
		name = "ENCAP"
	case 99:
		name = "Private Encryption"
	case 100:
		name = "GMTP"
	case 101:
		name = "IFMP"
	case 102:
		name = "PNNI"
	case 103:
		name = "PIM"
	case 104:
		name = "ARIS"
	case 105:
		name = "SCPS"
	case 106:
		name = "QNX"
	case 107:
		name = "A/N"
	case 108:
		name = "IPComp"
	case 109:
		name = "SNP"
	case 110:
		name = "Compaq-Peer"
	case 111:
		name = "IPX-in-IP"
	case 112:
		name = "VRRP"
	case 113:
		name = "PGM"
	case 114:
		name = "0-hop Protocol"
	case 115:
		name = "L2TP"
	case 116:
		name = "DDX"
	case 117:
		name = "IATP"
	case 118:
		name = "STP"
	case 119:
		name = "SRP"
	case 120:
		name = "UTI"
	case 121:
		name = "SMP"
	case 122:
		name = "SM"
	case 123:
		name = "PTP"
	case 124:
		name = "IS-IS over IPv4"
	case 125:
		name = "FIRE"
	case 126:
		name = "CRTP"
	case 127:
		name = "CRUDP"
	case 128:
		name = "SSCOPMCE"
	case 129:
		name = "IPLT"
	case 130:
		name = "SPS"
	case 131:
		name = "PIPE"
	case 132:
		name = "SCTP"
	case 133:
		name = "FC"
	case 134:
		name = "RSVP-E2E-IGNORE"
	case 135:
		name = "Mobility Header"
	case 136:
		name = "UDPLite"
	case 137:
		name = "MPLS-in-IP"
	case 138:
		name = "manet"
	case 139:
		name = "HIP"
	case 140:
		name = "Shim6"
	case 141:
		name = "WESP"
	case 142:
		name = "ROHC"
	case 143:
		name = "Ethernet"
	}

	return name
}
