package web

import (
	"math"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/stretchr/testify/assert"
)

func TestXPTLWithCfgClampsConsistentlyWithCharacter(t *testing.T) {
	const highLevel = 200_000_000

	cfg := configs.GetProgressionConfig()
	previewXP := xpTLWithCfg(highLevel, cfg)
	engineXP := characters.New().XPTL(highLevel)

	assert.Equal(t, math.MaxInt, previewXP)
	assert.Equal(t, engineXP, previewXP)
}
