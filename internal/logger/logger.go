package logger

import (
	"log"
	"os"
	"strings"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

var currentLevel Level = INFO

// InitLogger inicializa o logger com o nível especificado
func InitLogger(level string) {
	switch strings.ToUpper(level) {
	case "DEBUG":
		currentLevel = DEBUG
	case "INFO":
		currentLevel = INFO
	case "WARN", "WARNING":
		currentLevel = WARN
	case "ERROR":
		currentLevel = ERROR
	default:
		currentLevel = INFO
	}
}

// Debug loga apenas em modo DEBUG
func Debug(format string, v ...interface{}) {
	if currentLevel <= DEBUG {
		log.Printf("[DEBUG] "+format, v...)
	}
}

// Info loga em INFO ou superior
func Info(format string, v ...interface{}) {
	if currentLevel <= INFO {
		log.Printf("[INFO] "+format, v...)
	}
}

// Warn loga em WARN ou superior
func Warn(format string, v ...interface{}) {
	if currentLevel <= WARN {
		log.Printf("[WARN] "+format, v...)
	}
}

// Error sempre loga (nível mais alto)
func Error(format string, v ...interface{}) {
	if currentLevel <= ERROR {
		log.Printf("[ERROR] "+format, v...)
	}
}

// Fatal loga e encerra o programa
func Fatal(v ...interface{}) {
	log.Fatal(v...)
}

// Fatalf loga com formato e encerra o programa
func Fatalf(format string, v ...interface{}) {
	log.Fatalf(format, v...)
}

// SetOutput permite redirecionar logs para um arquivo (opcional)
func SetOutput(file *os.File) {
	log.SetOutput(file)
}

