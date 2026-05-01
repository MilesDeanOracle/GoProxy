package platform

import "testing"

func TestConfigAndLogPathUseExecutableDirectory(t *testing.T) {
	previous := executablePath
	executablePath = func() (string, error) {
		return `D:\goproxy\GoProxy.exe`, nil
	}
	defer func() {
		executablePath = previous
	}()

	configPath, err := ConfigPath()
	if err != nil {
		t.Fatalf("config path: %v", err)
	}
	if want := `D:\goproxy\configs\config.yaml`; configPath != want {
		t.Fatalf("expected config path %q, got %q", want, configPath)
	}

	logPath, err := LogPath()
	if err != nil {
		t.Fatalf("log path: %v", err)
	}
	if want := `D:\goproxy\logs\proxy-server.log`; logPath != want {
		t.Fatalf("expected log path %q, got %q", want, logPath)
	}
}
