package maker

import (
	"testing"
)

var (
	FilePath = ""
	QW       *QQwry
)

func TestMain(m *testing.M) {
	QW = NewQQwry(FilePath)
	m.Run()
}
