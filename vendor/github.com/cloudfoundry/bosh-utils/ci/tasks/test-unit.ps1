trap {
  write-error $_
  exit 1
}

$env:GOPATH = Join-Path -Path $PWD "gopath"
$env:PATH = $env:GOPATH + "/bin;C:/bin;" + $env:PATH

cd $env:GOPATH/src/github.com/cloudfoundry/bosh-utils

if ((Get-Command "tar.exe" -ErrorAction SilentlyContinue) -eq $null)
{
  Write-Host "Installing tar!"
  New-Item -ItemType directory -Path C:\bin -Force

  [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
  Invoke-WebRequest 'https://s3.amazonaws.com/bosh-windows-dependencies/tar-1490035387.exe' -OutFile C:\bin\tar.exe

  Write-Host "tar is installed!"
}

go.exe version

go.exe install github.com/cloudfoundry/bosh-utils/vendor/github.com/onsi/ginkgo/ginkgo
ginkgo.exe -r -keepGoing -skipPackage="vendor"
if ($LastExitCode -ne 0)
{
    Write-Error $_
    exit 1
}
