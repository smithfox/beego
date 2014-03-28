package context

import (
	"github.com/gorilla/schema"
)

var gGorillaDecoder *schema.Decoder

func init() {
	gGorillaDecoder = schema.NewDecoder()
	gGorillaDecoder.IgnoreUnknownKeys(true)
}
