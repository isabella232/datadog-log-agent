// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2017 Datadog, Inc.

package message

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMessage(t *testing.T) {
	message := NewMessage([]byte("hello"))
	assert.Equal(t, "hello", string(message.Content()))

	message.SetContent([]byte("world"))
	assert.Equal(t, "world", string(message.Content()))
}
