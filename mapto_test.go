package mapto

import (
	"time"
	"testing"
	"github.com/magiconair/properties/assert"
)

type TestCommonStruct struct {
	Dur time.Duration
	Val string
	Num int
}

type TestInterface interface{
	String() string
}

type TestNestedStruct struct {
	NestedVal string
}

func (tns * TestNestedStruct) String() string {
	return "TestNestedStruct:"+tns.NestedVal
}

type TestStruct struct {
	TestCommonStruct `mapstructure:",squash"`
	Dur time.Duration
	Val2 string
	Nested TestInterface
}

func TestDecode(t *testing.T) {
	m := map[string]interface{} {
		"dur": "2s",
		"val": "testvalue",
		"num": 42,
		"val2": "testvalue2",
		"nested": map[string]interface{}{ "nestedVal": "testNestedValue", "@type": "teststruct"},
	}
	RegisterConstructor("teststruct", StructConstructor(&TestNestedStruct{}))
	teststruct := TestStruct{}
	err := Decode(m, &teststruct)
	if err != nil {
		t.Error(err)
		return
	}

	assert.Equal(t, 2 * time.Second, teststruct.Dur)
	assert.Equal(t, 2 * time.Second, teststruct.TestCommonStruct.Dur)
	assert.Equal(t, "testvalue", teststruct.Val)
	assert.Equal(t, 42, teststruct.Num)
	assert.Equal(t, "testvalue2", teststruct.Val2)
	assert.Equal(t, "TestNestedStruct:testNestedValue", teststruct.Nested.String())
}
