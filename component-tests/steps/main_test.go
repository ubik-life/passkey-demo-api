package steps

import (
	"os"
	"testing"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
)

var opts = godog.Options{
	Output:    colors.Colored(os.Stdout),
	Format:    "pretty",
	Randomize: -1, // детерминированный порядок (для воспроизводимости)
}

func init() {
	godog.BindCommandLineFlags("godog.", &opts)
}

// TestFeatures запускает все .feature-файлы из ../features.
// Контейнер runner запускает: `go test -v -count=1 ./steps/...`
func TestFeatures(t *testing.T) {
	opts.Paths = []string{"../features"}

	status := godog.TestSuite{
		Name:                "passkey-demo-api",
		ScenarioInitializer: InitializeScenario,
		Options:             &opts,
	}.Run()

	if status != 0 {
		t.Fail()
	}
}

// InitializeScenario регистрирует все степ-дефиниции и lifecycle-хуки.
// Каждый сценарий получает свой World — состояние не утекает между сценариями.
func InitializeScenario(ctx *godog.ScenarioContext) {
	w := newWorld()

	ctx.Before(w.beforeScenario)
	ctx.After(w.afterScenario)

	w.registerHTTPSteps(ctx)
	w.registerWebAuthnSteps(ctx)
	w.registerDBFailureSteps(ctx)
	w.registerAuthSteps(ctx)
}
