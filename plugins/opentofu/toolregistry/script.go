package toolregistry

var OpenTofuInstallScript = `
cd {{ .TmpDir }}
curl -L https://github.com/opentofu/opentofu/releases/download/v{{ .Version }}/tofu_{{ .Version }}_{{ .Os }}_{{ .Arch }}.zip -o tofu_{{ .Version }}_{{ .Os }}_{{ .Arch }}.zip
unzip tofu_{{ .Version }}_{{ .Os }}_{{ .Arch }}.zip
mv tofu {{ .OutPath }}
`
