package logging

import (
	"log/slog"
	"os"
)

var L *slog.Logger // глобальный, но можно передавать контекстом

func Init() {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo, // через ENV можно переключать
	})
	L = slog.New(handler)
}