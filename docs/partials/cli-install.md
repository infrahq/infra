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

{% tab label="Ubuntu & Debian" %}
Download the [latest][1] Debian package from GitHub and install it with `dpkg` or `apt`.
```
sudo dpkg -i infra_*.deb
```
```
sudo apt install ./infra_*.deb
```
{% /tab %}
{% tab label="Fedora & RHEL" %}
Download the [latest][1] RPM package from GitHub and install it with `rpm` or `dnf`.
```
sudo rpm -i infra-*.rpm
```
```
sudo dnf install infra-*.rpm
```
{% /tab %}
{% tab label="Manual" %}
Download the [latest][1] release from GitHub, unpack the file, and add the binary to the `PATH`.
{% /tab %}
{% /tabs %}

[1]: https://github.com/infrahq/infra/releases/latest
