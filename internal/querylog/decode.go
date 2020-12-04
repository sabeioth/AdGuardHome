package querylog

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/dnsfilter"
	"github.com/AdguardTeam/golibs/log"
	"github.com/miekg/dns"
)

type logEntryHandler (func(t json.Token, ent *logEntry) error)

var logEntryHandlers = map[string]logEntryHandler{
	"IP": func(t json.Token, ent *logEntry) error {
		v, ok := t.(string)
		if !ok {
			return nil
		}
		if len(ent.IP) == 0 {
			ent.IP = v
		}
		return nil
	},
	"T": func(t json.Token, ent *logEntry) error {
		v, ok := t.(string)
		if !ok {
			return nil
		}
		var err error
		ent.Time, err = time.Parse(time.RFC3339, v)
		return err
	},
	"QH": func(t json.Token, ent *logEntry) error {
		v, ok := t.(string)
		if !ok {
			return nil
		}
		ent.QHost = v
		return nil
	},
	"QT": func(t json.Token, ent *logEntry) error {
		v, ok := t.(string)
		if !ok {
			return nil
		}
		ent.QType = v
		return nil
	},
	"QC": func(t json.Token, ent *logEntry) error {
		v, ok := t.(string)
		if !ok {
			return nil
		}
		ent.QClass = v
		return nil
	},
	"CP": func(t json.Token, ent *logEntry) error {
		v, ok := t.(string)
		if !ok {
			return nil
		}
		var err error
		ent.ClientProto, err = NewClientProto(v)
		return err
	},
	"Answer": func(t json.Token, ent *logEntry) error {
		v, ok := t.(string)
		if !ok {
			return nil
		}
		var err error
		ent.Answer, err = base64.StdEncoding.DecodeString(v)
		return err
	},
	"OrigAnswer": func(t json.Token, ent *logEntry) error {
		v, ok := t.(string)
		if !ok {
			return nil
		}
		var err error
		ent.OrigAnswer, err = base64.StdEncoding.DecodeString(v)
		return err
	},
	"IsFiltered": func(t json.Token, ent *logEntry) error {
		v, ok := t.(bool)
		if !ok {
			return nil
		}
		ent.Result.IsFiltered = v
		return nil
	},
	"Rule": func(t json.Token, ent *logEntry) error {
		v, ok := t.(string)
		if !ok {
			return nil
		}
		ent.Result.Rule = v
		return nil
	},
	"FilterID": func(t json.Token, ent *logEntry) error {
		v, ok := t.(json.Number)
		if !ok {
			return nil
		}
		i, err := v.Int64()
		if err != nil {
			return err
		}
		ent.Result.FilterID = i
		return nil
	},
	"Reason": func(t json.Token, ent *logEntry) error {
		v, ok := t.(json.Number)
		if !ok {
			return nil
		}
		i, err := v.Int64()
		if err != nil {
			return err
		}
		ent.Result.Reason = dnsfilter.Reason(i)
		return nil
	},
	"ServiceName": func(t json.Token, ent *logEntry) error {
		v, ok := t.(string)
		if !ok {
			return nil
		}
		ent.Result.ServiceName = v
		return nil
	},
	"Upstream": func(t json.Token, ent *logEntry) error {
		v, ok := t.(string)
		if !ok {
			return nil
		}
		ent.Upstream = v
		return nil
	},
	"Elapsed": func(t json.Token, ent *logEntry) error {
		v, ok := t.(json.Number)
		if !ok {
			return nil
		}
		i, err := v.Int64()
		if err != nil {
			return err
		}
		ent.Elapsed = time.Duration(i)
		return nil
	},
	"Result": func(json.Token, *logEntry) error {
		return nil
	},
	"Question": func(t json.Token, ent *logEntry) error {
		v, ok := t.(string)
		if !ok {
			return nil
		}
		var qstr []byte
		qstr, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			return err
		}
		q := new(dns.Msg)
		err = q.Unpack(qstr)
		if err != nil {
			return err
		}
		ent.QHost = q.Question[0].Name
		if len(ent.QHost) == 0 {
			return nil // nil???
		}
		ent.QHost = ent.QHost[:len(ent.QHost)-1]
		ent.QType = dns.TypeToString[q.Question[0].Qtype]
		ent.QClass = dns.ClassToString[q.Question[0].Qclass]
		return nil
	},
	"Time": func(t json.Token, ent *logEntry) error {
		v, ok := t.(string)
		if !ok {
			return nil
		}
		var err error
		ent.Time, err = time.Parse(time.RFC3339, v)
		return err
	},
}

func decodeLogEntry(ent *logEntry, str string) {
	dec := json.NewDecoder(strings.NewReader(str))
	dec.UseNumber()
	for {
		keyToken, err := dec.Token()
		if err != nil {
			if err != io.EOF {
				log.Debug("decodeLogEntry err: %s", err)
			}
			return
		}
		if _, ok := keyToken.(json.Delim); ok {
			continue
		}

		key, ok := keyToken.(string)
		if !ok {
			log.Debug("decodeLogEntry: keyToken is %T and not string", keyToken)
			return
		}

		handler, ok := logEntryHandlers[key]
		if !ok {
			continue
		}
		val, err := dec.Token()
		if err != nil {
			return
		}

		if ok {
			if err := handler(val, ent); err != nil {
				log.Debug("decodeLogEntry err: %s", err)
				return
			}
		}
	}
}

// Get value from "key":"value"
func readJSONValue(s, name string) string {
	i := strings.Index(s, "\""+name+"\":\"")
	if i == -1 {
		return ""
	}
	start := i + 1 + len(name) + 3
	i = strings.IndexByte(s[start:], '"')
	if i == -1 {
		return ""
	}
	end := start + i
	return s[start:end]
}
