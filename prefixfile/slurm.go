// rfc8416 and draft-sidrops-aspa-slurm

package prefixfile

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io"
	"net"
	"strings"
)

type SlurmPrefixFilter struct {
	Prefix  string
	ASN     *uint32 `json:"asn,omitempty"`
	Comment string
}

type SlurmBGPsecFilter struct {
	ASN     *uint32 `json:"asn,omitempty"`
	SKI     []byte  `json:"SKI,omitempty"`
	Comment string  `json:"comment"`
}

type SlurmASPAFilter struct {
	Afi          string `json:"afi"`
	Comment      string `json:"comment"`
	CustomerASid uint32 `json:"customer_asid"`
}

func (pf *SlurmPrefixFilter) GetASN() (uint32, bool) {
	if pf.ASN == nil {
		return 0, true
	} else {
		return *pf.ASN, false
	}
}

func (pf *SlurmPrefixFilter) GetPrefix() *net.IPNet {
	_, prefix, _ := net.ParseCIDR(pf.Prefix)
	return prefix
}

type SlurmValidationOutputFilters struct {
	PrefixFilters []SlurmPrefixFilter
	BgpsecFilters []SlurmBGPsecFilter
	AspaFilters   []SlurmASPAFilter
}

type SlurmPrefixAssertion struct {
	Prefix          string
	ASN             uint32
	MaxPrefixLength int
	Comment         string
}

type SlurmBGPsecAssertion struct {
	SKI             []byte `json:"SKI"`
	ASN             uint32 `json:"asn"`
	Comment         string `json:"comment"`
	RouterPublicKey []byte `json:"routerPublicKey"`
}

type SlurmASPAAssertion struct {
	Afi           string   `json:"afi"`
	Comment       string   `json:"comment"`
	CustomerASNid uint32   `json:"customer_asid"`
	ProviderSet   []uint32 `json:"provider_set"`
}

func (pa *SlurmPrefixAssertion) GetASN() uint32 {
	return pa.ASN
}

func (pa *SlurmPrefixAssertion) GetPrefix() *net.IPNet {
	_, prefix, _ := net.ParseCIDR(pa.Prefix)
	return prefix
}

func (pa *SlurmPrefixAssertion) GetMaxLen() int {
	return pa.MaxPrefixLength
}

type SlurmLocallyAddedAssertions struct {
	PrefixAssertions []SlurmPrefixAssertion
	BgpsecAssertions []SlurmBGPsecAssertion
	AspaAssertions   []SlurmASPAAssertion
}

type SlurmConfig struct {
	SlurmVersion            int
	ValidationOutputFilters SlurmValidationOutputFilters
	LocallyAddedAssertions  SlurmLocallyAddedAssertions
}

func DecodeJSONSlurm(buf io.Reader) (*SlurmConfig, error) {
	slurm := &SlurmConfig{}
	dec := json.NewDecoder(buf)
	dec.UseNumber()
	err := dec.Decode(slurm)
	if err != nil {
		return nil, err
	}
	return slurm, nil
}

func (s *SlurmValidationOutputFilters) FilterOnVRPs(vrps []VRPJson) (added, removed []VRPJson) {
	added = make([]VRPJson, 0)
	removed = make([]VRPJson, 0)
	if s.PrefixFilters == nil || len(s.PrefixFilters) == 0 {
		return vrps, removed
	}
	for _, vrp := range vrps {
		rPrefix := vrp.GetPrefix()
		var rIPStart net.IP
		var rIPEnd net.IP
		if rPrefix != nil {
			rIPStart = rPrefix.IP.To16()
			rIPEnd = GetIPBroadcast(*rPrefix).To16()
		}

		var wasRemoved bool
		for _, filter := range s.PrefixFilters {
			fPrefix := filter.GetPrefix()
			fASN, fASNEmpty := filter.GetASN()
			match := true
			if match && fPrefix != nil && rPrefix != nil {

				if !(fPrefix.Contains(rIPStart) && fPrefix.Contains(rIPEnd)) {
					match = false
				}
			}
			if match && !fASNEmpty {
				if vrp.GetASN() != fASN {
					match = false
				}
			}
			if match {
				removed = append(removed, vrp)
				wasRemoved = true
				break
			}
		}

		if !wasRemoved {
			added = append(added, vrp)
		}
	}
	return added, removed
}

func (s *SlurmValidationOutputFilters) FilterOnBRKs(brks []BgpSecKeyJson) (added, removed []BgpSecKeyJson) {
	added = make([]BgpSecKeyJson, 0)
	removed = make([]BgpSecKeyJson, 0)
	if s.BgpsecFilters == nil || len(s.BgpsecFilters) == 0 {
		return brks, removed
	}
	for _, brk := range brks {
		var skiCache []byte
		var wasRemoved bool
		for _, filter := range s.BgpsecFilters {
			if filter.ASN != nil {
				if brk.Asn == *filter.ASN {
					if len(filter.SKI) != 0 {
						// We need to compare the SKIs then
						if skiCache == nil { // We have not yet decoded the ski hex
							var err error
							skiCache, err = hex.DecodeString(brk.Ski)
							if err != nil {
								// Ski could not be decoded, so we can't filter
								continue
							}
						}
						if bytes.Equal(filter.SKI, skiCache) {
							removed = append(removed, brk)
							wasRemoved = true
							break
						}
					} else {
						// Only a ASN match was needed
						removed = append(removed, brk)
						wasRemoved = true
						break
					}
				}
			}

			if len(filter.SKI) != 0 && filter.ASN == nil {
				// We need to compare just the SKIs then
				if skiCache == nil { // We have not yet decoded the ski hex
					var err error
					skiCache, err = hex.DecodeString(brk.Ski)
					if err != nil {
						// Ski could not be decoded, so we can't filter
						continue
					}
				}
				if bytes.Equal(filter.SKI, skiCache) {
					removed = append(removed, brk)
					wasRemoved = true
					break
				}
			}
		}

		if !wasRemoved {
			added = append(added, brk)
		}
	}
	return added, removed
}

func (s *SlurmValidationOutputFilters) FilterOnVAPs(vaps []ASPAJson, ipv6 bool) (added, removed []ASPAJson) {
	added = make([]ASPAJson, 0)
	removed = make([]ASPAJson, 0)
	if s.AspaFilters == nil || len(s.AspaFilters) == 0 {
		return vaps, removed
	}
	for _, vap := range vaps {
		var wasRemoved bool
		for _, filter := range s.AspaFilters {
			if strings.Contains(filter.Afi, "6") && !ipv6 {
				continue
			}

			if vap.CustomerAsid == filter.CustomerASid {
				removed = append(removed, vap)
				wasRemoved = true
				break
			}
		}

		if !wasRemoved {
			added = append(added, vap)
		}
	}
	return added, removed
}

func (s *SlurmLocallyAddedAssertions) AssertVRPs() []VRPJson {
	vrps := make([]VRPJson, 0)
	if s.PrefixAssertions == nil || len(s.PrefixAssertions) == 0 {
		return vrps
	}
	for _, assertion := range s.PrefixAssertions {
		prefix := assertion.GetPrefix()
		if prefix == nil {
			continue
		}
		size, _ := prefix.Mask.Size()
		maxLength := assertion.MaxPrefixLength
		if assertion.MaxPrefixLength <= size {
			maxLength = size
		}
		vrps = append(vrps, VRPJson{
			ASN:    uint32(assertion.ASN),
			Prefix: assertion.Prefix,
			Length: uint8(maxLength),
			TA:     assertion.Comment,
		})
	}
	return vrps
}

func (s *SlurmLocallyAddedAssertions) AssertVAPs() (v4, v6 []ASPAJson) {
	vapsv4, vapsv6 := make([]ASPAJson, 0), make([]ASPAJson, 0)

	if s.AspaAssertions == nil || len(s.AspaAssertions) == 0 {
		return vapsv4, vapsv6
	}
	for _, assertion := range s.AspaAssertions {
		vap := ASPAJson{
			CustomerAsid: assertion.CustomerASNid,
			Providers:    assertion.ProviderSet,
		}
		if strings.Contains(assertion.Afi, "6") {
			vapsv6 = append(vapsv6, vap)
		} else {
			vapsv4 = append(vapsv4, vap)
		}
	}
	return vapsv4, vapsv6
}

func (s *SlurmLocallyAddedAssertions) AssertBRKs() []BgpSecKeyJson {
	brks := make([]BgpSecKeyJson, 0)

	if s.BgpsecAssertions == nil || len(s.BgpsecAssertions) == 0 {
		return brks
	}
	for _, assertion := range s.BgpsecAssertions {
		hexSki := hex.EncodeToString(assertion.SKI)
		brk := BgpSecKeyJson{
			Asn:    assertion.ASN,
			Pubkey: assertion.RouterPublicKey,
			Ski:    hexSki,
		}
		brks = append(brks, brk)
	}
	return brks
}

func (s *SlurmConfig) GetAssertions() (vrps []VRPJson, VAPv4, VAPv6 []ASPAJson, BRKs []BgpSecKeyJson) {
	vrps = s.LocallyAddedAssertions.AssertVRPs()
	VAPv4, VAPv6 = s.LocallyAddedAssertions.AssertVAPs()
	BRKs = s.LocallyAddedAssertions.AssertBRKs()
	return
}

func (s *SlurmConfig) FilterAssert(vrps []VRPJson, VAPv4, VAPv6 []ASPAJson, BRKs []BgpSecKeyJson, log Logger) (
	ovrps []VRPJson, oVAPv4, oVAPv6 []ASPAJson, oBRKs []BgpSecKeyJson) {
	//
	filteredVRPs, removedVRPs := s.ValidationOutputFilters.FilterOnVRPs(vrps)
	filteredVAP4s, removedVAP4s := s.ValidationOutputFilters.FilterOnVAPs(VAPv4, false)
	filteredVAP6s, removedVAP6s := s.ValidationOutputFilters.FilterOnVAPs(VAPv6, true)
	filteredBRKs, removedBRKs := s.ValidationOutputFilters.FilterOnBRKs(BRKs)

	assertVRPs, assertVAP4, assertVAP6, assertBRKs := s.GetAssertions()

	ovrps = append(filteredVRPs, assertVRPs...)
	oVAPv4 = append(filteredVAP4s, assertVAP4...)
	oVAPv6 = append(filteredVAP6s, assertVAP6...)
	oBRKs = append(filteredBRKs, assertBRKs...)

	if log != nil {
		if len(s.ValidationOutputFilters.PrefixFilters) != 0 {
			log.Infof("Slurm VRP filtering: %v kept, %v removed, %v asserted", len(filteredVRPs), len(removedVRPs), len(ovrps))
		}

		if len(s.ValidationOutputFilters.BgpsecFilters) != 0 {
			log.Infof("Slurm Router Key filtering: %v kept, %v removed, %v asserted", len(filteredBRKs), len(removedBRKs), len(oBRKs))
		}

		if len(s.ValidationOutputFilters.AspaFilters) != 0 {
			log.Infof("Slurm ASPA v4 filtering: %v kept, %v removed, %v asserted", len(filteredVAP4s), len(removedVAP4s), len(oVAPv4))
			log.Infof("Slurm ASPA v6 filtering: %v kept, %v removed, %v asserted", len(filteredVAP6s), len(removedVAP6s), len(oVAPv6))
		}
	}
	return
}

type Logger interface {
	Debugf(string, ...interface{})
	Printf(string, ...interface{})
	Warnf(string, ...interface{})
	Errorf(string, ...interface{})
	Infof(string, ...interface{})
}
