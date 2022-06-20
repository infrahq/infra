{% tabs %}
{% tab label="macOS" %}
```
brew install infrahq/tap/infra
```
{% /tab %}
{% tab label="Windows" %}
```powershell
scoop bucket add infrahq https://github.com/infrahq/scoop.git
scoop install infra
```
{% /tab %}

{% tab label="Linux" %}

#### Ubuntu & Debian
```
echo 'deb [trusted=yes] https://apt.fury.io/infrahq/ /' | sudo tee /etc/apt/sources.list.d/infrahq.list
sudo apt update
sudo apt install infra
```

#### Fedora & Red Hat Enterprise Linux
```
sudo dnf config-manager --add-repo https://yum.fury.io/infrahq/
sudo dnf install infra
```
{% /tab %}
{% /tabs %}
