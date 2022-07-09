package logger

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

func newEncoder() *toolshareEncoder {
	return &toolshareEncoder{
		Encoder:       zapcore.NewConsoleEncoder(fieldEncoderConfig),
		EncoderConfig: externalEncoderConfig,
	}
}

type toolshareEncoder struct {
	zapcore.Encoder
	zapcore.EncoderConfig
}

func (c *toolshareEncoder) Clone() zapcore.Encoder {
	return &toolshareEncoder{
		Encoder:       c.Encoder.Clone(),
		EncoderConfig: c.EncoderConfig,
	}
}

func (c toolshareEncoder) EncodeEntry(ent zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	line := pool.Get()

	arr := &sliceArrayEncoder{}
	c.EncodeTime(ent.Time, arr)
	c.EncodeLevel(ent.Level, arr)
	c.EncodeName(ent.LoggerName, arr)
	if ent.Caller.Defined && c.EncodeCaller != nil {
		c.EncodeCaller(ent.Caller, arr)
	}
	arr.AppendString(ent.Message)

	for i := range arr.elems {
		if i > 0 {
			line.AppendByte(' ')
		}
		fmt.Fprint(line, arr.elems[i])
	}

	if ent.Level == zapcore.InfoLevel {
		// We do not output fields on info-level as standard user output should be as readable as
		// possible. Other levels should add as much information as possible to provide context.
		line.AppendByte('\n')
		return line, nil
	}

	b, err := c.Encoder.EncodeEntry(zapcore.Entry{}, fields)
	if err != nil {
		return nil, err
	}
	buf := bytes.TrimSpace(b.Bytes())

	if len(buf) > 0 {
		if _, err = line.WriteString(fieldPrefix + string(buf) + "\n"); err != nil {
			return nil, err
		}
	} else {
		line.AppendString("\n")
	}
	return line, nil
}

var (
	pool = buffer.NewPool()

	externalEncoderConfig = zapcore.EncoderConfig{
		LevelKey:       "level",
		TimeKey:        "time",
		MessageKey:     "msg",
		CallerKey:      "caller",
		EncodeLevel:    levelEncoder,
		EncodeTime:     func(t time.Time, enc zapcore.PrimitiveArrayEncoder) { enc.AppendString(t.Format("15:04:05")) },
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.FullCallerEncoder,
		EncodeName:     nameEncoder,
	}
	fieldEncoderConfig = zapcore.EncoderConfig{
		// We omit any of the keys so that we don't print any of the already previously printed fields.
		EncodeLevel:    zapcore.LowercaseColorLevelEncoder,
		EncodeTime:     func(t time.Time, enc zapcore.PrimitiveArrayEncoder) { enc.AppendString(t.Format("15:04:05")) },
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.FullCallerEncoder,
	}

	levelToColor = map[zapcore.Level]*color.Color{
		zapcore.DPanicLevel: color.New(color.FgHiRed),
		zapcore.PanicLevel:  color.New(color.FgHiRed),
		zapcore.FatalLevel:  color.New(color.FgRed),
		zapcore.ErrorLevel:  color.New(color.FgRed),
		zapcore.WarnLevel:   color.New(color.FgYellow),
		zapcore.InfoLevel:   color.New(color.FgBlue),
		zapcore.DebugLevel:  color.New(color.FgMagenta),
	}
)

func levelEncoder(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(levelToColor[l].Sprintf("%-7s", l))
}

var (
	nameEncoderPattern string
	fieldPrefix        string
)

func init() {
	var l int
	for n := range domainFromString {
		if l < len(n) {
			l = len(n)
		}
	}
	nameEncoderPattern = fmt.Sprintf("%%-%ds", l+1)
	fieldPrefix = fmt.Sprintf("\n%s", strings.Repeat(" ", 18+l)) // 9 for time, 7 for level, variable for name-encoding.
}

func nameEncoder(name string, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(fmt.Sprintf(nameEncoderPattern, name))
}

type sliceArrayEncoder struct {
	elems []interface{}
}

func (s *sliceArrayEncoder) AppendArray(v zapcore.ArrayMarshaler) error {
	enc := &sliceArrayEncoder{}
	err := v.MarshalLogArray(enc)
	s.elems = append(s.elems, enc.elems)
	return err
}

func (s *sliceArrayEncoder) AppendObject(v zapcore.ObjectMarshaler) error {
	m := zapcore.NewMapObjectEncoder()
	err := v.MarshalLogObject(m)
	s.elems = append(s.elems, m.Fields)
	return err
}

func (s *sliceArrayEncoder) AppendReflected(v interface{}) error {
	s.elems = append(s.elems, v)
	return nil
}

func (s *sliceArrayEncoder) AppendBool(v bool)              { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendByteString(v []byte)      { s.elems = append(s.elems, string(v)) }
func (s *sliceArrayEncoder) AppendComplex128(v complex128)  { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendComplex64(v complex64)    { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendDuration(v time.Duration) { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendFloat64(v float64)        { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendFloat32(v float32)        { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendInt(v int)                { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendInt64(v int64)            { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendInt32(v int32)            { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendInt16(v int16)            { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendInt8(v int8)              { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendString(v string)          { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendTime(v time.Time)         { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendUint(v uint)              { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendUint64(v uint64)          { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendUint32(v uint32)          { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendUint16(v uint16)          { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendUint8(v uint8)            { s.elems = append(s.elems, v) }
func (s *sliceArrayEncoder) AppendUintptr(v uintptr)        { s.elems = append(s.elems, v) }
