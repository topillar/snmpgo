package snmpgo

import (
	"encoding/asn1"
	"fmt"
	"strings"
)

type VarBind struct {
	Oid      *Oid
	Variable Variable
}

func (v *VarBind) Marshal() (b []byte, err error) {
	var buf []byte
	raw := asn1.RawValue{Class: classUniversal, Tag: tagSequence, IsCompound: true}

	if v.Oid == nil || v.Variable == nil {
		return asn1.Marshal(raw)
	}

	buf, err = v.Oid.Marshal()
	if err != nil {
		return
	}
	raw.Bytes = buf

	buf, err = v.Variable.Marshal()
	if err != nil {
		return
	}
	raw.Bytes = append(raw.Bytes, buf...)

	return asn1.Marshal(raw)
}

func (v *VarBind) Unmarshal(b []byte) (rest []byte, err error) {
	var raw asn1.RawValue
	rest, err = asn1.Unmarshal(b, &raw)
	if err != nil {
		return nil, err
	}
	if raw.Class != classUniversal || raw.Tag != tagSequence || !raw.IsCompound {
		return nil, asn1.StructuralError{fmt.Sprintf(
			"Invalid VarBind object - Class [%02x], Tag [%02x] : [%s]",
			raw.Class, raw.Tag, toHexStr(b, " "))}
	}

	var oid Oid
	next, err := (&oid).Unmarshal(raw.Bytes)
	if err != nil {
		return
	}

	variable, _, err := unmarshalVariable(next)
	if err != nil {
		return
	}

	v.Oid = &oid
	v.Variable = variable
	return
}

func (v *VarBind) String() string {
	var oid, vtype, value string
	if v.Oid != nil {
		oid = v.Oid.String()
	}
	if v.Variable != nil {
		vtype = v.Variable.Type()
		value = escape(v.Variable.String())
	}
	return fmt.Sprintf(`{"Oid": "%s", "Variable": {"Type": "%s", "Value": %s}}`,
		oid, vtype, value)
}

type VarBinds []*VarBind

func (v VarBinds) MatchOid(oid *Oid) *VarBind {
	for _, o := range v {
		if o.Oid != nil && o.Oid.Equal(oid) {
			return o
		}
	}
	return nil
}

func (v VarBinds) BaseMatchOids(oid *Oid) VarBinds {
	result := make(VarBinds, 0)
	for _, o := range v {
		if o.Oid != nil && o.Oid.Contains(oid) {
			result = append(result, o)
		}
	}
	return result
}

func (v VarBinds) String() string {
	varBinds := make([]string, len(v))
	for i, o := range v {
		varBinds[i] = o.String()
	}
	return "[" + strings.Join(varBinds, ", ") + "]"
}

type Pdu interface {
	PduType() PduType
	RequestId() int
	SetRequestId(int)
	ErrorStatus() ErrorStatus
	SetErrorStatus(ErrorStatus)
	ErrorIndex() int
	SetErrorIndex(int)
	SetNonrepeaters(int)
	SetMaxRepetitions(int)
	AppendVarBind(*Oid, Variable)
	VarBinds() VarBinds
	Marshal() ([]byte, error)
	Unmarshal([]byte) (rest []byte, err error)
	String() string
}

type PduV1 struct {
	pduType     PduType
	requestId   int
	errorStatus ErrorStatus
	errorIndex  int
	varBinds    VarBinds
}

func (pdu *PduV1) PduType() PduType {
	return pdu.pduType
}

func (pdu *PduV1) RequestId() int {
	return pdu.requestId
}

func (pdu *PduV1) SetRequestId(i int) {
	pdu.requestId = i
}

func (pdu *PduV1) ErrorStatus() ErrorStatus {
	return pdu.errorStatus
}

func (pdu *PduV1) SetErrorStatus(i ErrorStatus) {
	pdu.errorStatus = i
}

func (pdu *PduV1) ErrorIndex() int {
	return pdu.errorIndex
}

func (pdu *PduV1) SetErrorIndex(i int) {
	pdu.errorIndex = i
}

func (pdu *PduV1) SetNonrepeaters(i int) {
	pdu.errorStatus = ErrorStatus(i)
}

func (pdu *PduV1) SetMaxRepetitions(i int) {
	pdu.errorIndex = i
}

func (pdu *PduV1) AppendVarBind(oid *Oid, variable Variable) {
	pdu.varBinds = append(pdu.varBinds, &VarBind{
		Oid:      oid,
		Variable: variable,
	})
}

func (pdu *PduV1) VarBinds() VarBinds {
	return pdu.varBinds
}

func (pdu *PduV1) Marshal() (b []byte, err error) {
	var buf []byte
	raw := asn1.RawValue{Class: classContextSpecific, Tag: int(pdu.pduType), IsCompound: true}

	buf, err = asn1.Marshal(pdu.requestId)
	if err != nil {
		return
	}
	raw.Bytes = buf

	buf, err = asn1.Marshal(pdu.errorStatus)
	if err != nil {
		return
	}
	raw.Bytes = append(raw.Bytes, buf...)

	buf, err = asn1.Marshal(pdu.errorIndex)
	if err != nil {
		return
	}
	raw.Bytes = append(raw.Bytes, buf...)

	varBinds := asn1.RawValue{Class: classUniversal, Tag: tagSequence, IsCompound: true}
	for i := 0; i < len(pdu.varBinds); i++ {
		buf, err = pdu.varBinds[i].Marshal()
		if err != nil {
			return
		}
		varBinds.Bytes = append(varBinds.Bytes, buf...)
	}

	buf, err = asn1.Marshal(varBinds)
	if err != nil {
		return
	}
	raw.Bytes = append(raw.Bytes, buf...)

	return asn1.Marshal(raw)
}

func (pdu *PduV1) Unmarshal(b []byte) (rest []byte, err error) {
	var raw asn1.RawValue
	rest, err = asn1.Unmarshal(b, &raw)
	if err != nil {
		return
	}
	if raw.Class != classContextSpecific || !raw.IsCompound {
		return nil, asn1.StructuralError{fmt.Sprintf(
			"Invalid Pdu object - Class [%02x], Tag [%02x] : [%s]",
			raw.Class, raw.Tag, toHexStr(b, " "))}
	}

	next := raw.Bytes

	var requestId int
	next, err = asn1.Unmarshal(next, &requestId)
	if err != nil {
		return
	}

	var errorStatus int
	next, err = asn1.Unmarshal(next, &errorStatus)
	if err != nil {
		return
	}

	var errorIndex int
	next, err = asn1.Unmarshal(next, &errorIndex)
	if err != nil {
		return
	}

	var varBinds asn1.RawValue
	_, err = asn1.Unmarshal(next, &varBinds)
	if err != nil {
		return
	}
	if varBinds.Class != classUniversal || varBinds.Tag != tagSequence || !varBinds.IsCompound {
		return nil, asn1.StructuralError{fmt.Sprintf(
			"Invalid VarBinds object - Class [%02x], Tag [%02x] : [%s]",
			varBinds.Class, varBinds.Tag, toHexStr(next, " "))}
	}

	next = varBinds.Bytes
	for len(next) > 0 {
		var varBind VarBind
		next, err = (&varBind).Unmarshal(next)
		if err != nil {
			return
		}
		pdu.varBinds = append(pdu.varBinds, &varBind)
	}

	pdu.pduType = PduType(raw.Tag)
	pdu.requestId = requestId
	pdu.errorStatus = ErrorStatus(errorStatus)
	pdu.errorIndex = errorIndex
	return
}

func (pdu *PduV1) String() string {
	return fmt.Sprintf(
		`{"Type": "%s", "RequestId": "%d", "ErrorStatus": "%s", `+
			`"ErrorIndex": "%d", "VarBinds": %s}`,
		pdu.pduType, pdu.requestId, pdu.errorStatus, pdu.errorIndex,
		pdu.varBinds.String())
}

type ScopedPdu struct {
	ContextEngineId []byte
	ContextName     []byte
	PduV1
}

func (pdu *ScopedPdu) Marshal() (b []byte, err error) {
	var buf []byte
	raw := asn1.RawValue{Class: classUniversal, Tag: tagSequence, IsCompound: true}

	buf, err = asn1.Marshal(pdu.ContextEngineId)
	if err != nil {
		return
	}
	raw.Bytes = buf

	buf, err = asn1.Marshal(pdu.ContextName)
	if err != nil {
		return
	}
	raw.Bytes = append(raw.Bytes, buf...)

	buf, err = pdu.PduV1.Marshal()
	if err != nil {
		return
	}
	raw.Bytes = append(raw.Bytes, buf...)

	return asn1.Marshal(raw)
}

func (pdu *ScopedPdu) Unmarshal(b []byte) (rest []byte, err error) {
	var raw asn1.RawValue
	rest, err = asn1.Unmarshal(b, &raw)
	if err != nil {
		return nil, err
	}
	if raw.Class != classUniversal || raw.Tag != tagSequence || !raw.IsCompound {
		return nil, asn1.StructuralError{fmt.Sprintf(
			"Invalid ScopedPud object - Class [%02x], Tag [%02x] : [%s]",
			raw.Class, raw.Tag, toHexStr(b, " "))}
	}

	next := raw.Bytes

	var contextEngineId []byte
	next, err = asn1.Unmarshal(next, &contextEngineId)
	if err != nil {
		return
	}

	var contextName []byte
	next, err = asn1.Unmarshal(next, &contextName)
	if err != nil {
		return
	}

	var pduV1 PduV1
	_, err = (&pduV1).Unmarshal(next)
	if err != nil {
		return
	}

	pdu.ContextEngineId = contextEngineId
	pdu.ContextName = contextName
	pdu.PduV1 = pduV1
	return
}

func (pdu *ScopedPdu) String() string {
	return fmt.Sprintf(
		`{"Type": "%s", "RequestId": "%d", "ErrorStatus": "%s", "ErrorIndex": "%d", `+
			`"ContextEngineId": "%s", "ContextName": %s, "VarBinds": %s}`,
		pdu.pduType, pdu.requestId, pdu.errorStatus, pdu.errorIndex,
		toHexStr(pdu.ContextEngineId, ""), escape(string(pdu.ContextName)),
		pdu.varBinds.String())
}

func NewPdu(ver SNMPVersion, t PduType) (pdu Pdu) {
	p := PduV1{pduType: t}
	switch ver {
	case V1, V2c:
		pdu = &p
	case V3:
		pdu = &ScopedPdu{PduV1: p}
	}
	return
}

func NewPduWithOids(ver SNMPVersion, t PduType, oids []Oid) (pdu Pdu) {
	pdu = NewPdu(ver, t)
	for i := 0; i < len(oids); i++ {
		pdu.AppendVarBind(&oids[i], NewNull())
	}
	return
}

func NewPduWithVarBinds(ver SNMPVersion, t PduType, varBinds VarBinds) (pdu Pdu) {
	pdu = NewPdu(ver, t)
	for _, v := range varBinds {
		pdu.AppendVarBind(v.Oid, v.Variable)
	}
	return
}