# SSH

> SSH is currently in early preview. We'd love to hear from you. [Contact us](mailto:contact@infrahq.com) or open a [GitHub issue](https://github.com/infrahq/infra/issues/new?labels=area/destinations/ssh) if you'd like to share feedback.

## Setup

This guide will walk you through installing the Infra connector on a Linux machine so that
you can connect to it with `ssh`.

### Login and enable SSH

On your desktop, [download the Infra CLI](../download.md) and log in to Infra. If you
don't yet have an Infra organization [signup here](https://signup.infrahq.com/).

```
infra login <your infra host> --enable-ssh
```

### Create a connector access key

```
infra keys add --connector
```

You will use this access key later in this guide.


### Install Infra

Install Infra on the Linux SSH host.

### Ubuntu & Debian

**Set up the repository**

```
curl -fsSL https://pkg.infrahq.com/apt/gpg.key | sudo gpg --dearmor -o /usr/share/keyrings/infra.gpg
sudo echo "deb [signed-by=/usr/share/keyrings/infra.gpg] https://pkg.infrahq.com/apt * *" | sudo tee /etc/apt/sources.list.d/infra.list > /dev/null
```

**Install Infra**

```
sudo apt-get update
sudo apt-get install infra
```

### Fedora & Red Hat

**Set up the repository**

```
cat << EOF | sudo tee /etc/yum.repos.d/infra.repo
[infra]
name=Infra
baseurl=https://pkg.infrahq.com/yum
enabled=1
type=rpm
repo_gpgcheck=1
gpgcheck=0
gpgkey=https://pkg.infrahq.com/yum/gpg.key
EOF
```

**Install Infra**

```
sudo yum install infra
```

### Amazon Linux

**Set up the repository**

```
cat << EOF | sudo tee /etc/yum.repos.d/infra.repo
[infra]
name=Infra
baseurl=https://pkg.infrahq.com/yum
enabled=1
type=rpm
repo_gpgcheck=1
gpgcheck=0
gpgkey=https://pkg.infrahq.com/yum/gpg.key
EOF
```

**Install Infra**

```
sudo yum install infra
```

### Other

**Install dependencies**

The following dependencies must be installed:

```
openssh-server
useradd
userdel
pkill
```

**Download the Infra binary**

Download the Infra binary from the latest [GitHub release](https://github.com/infrahq/infra/releases/latest) and place it in `/usr/local/sbin/infra`

> Note: the `infra` binary and the directory that contains the binary must be owned by root and not group or other writeable.

**Add the `systemd` service files**

```
sudo curl https://raw.githubusercontent.com/infrahq/infra/main/package-files/systemd/infra.service > /usr/lib/systemd/system/infra.service
```

**Add required users and groups**

```
sudo groupadd --system infra
sudo useradd --system --shell "$NOLOGIN" --home-dir "$HOMEDIR" --no-create-home --no-user-group --comment 'Infra Agent' infra
sudo usermod --group infra infra
sudo usermod --lock infra
sudo groupadd infra-users
```
  
### Setup and start the Infra connector

Next, on the Linux SSH host, create a configuration file with the access key
you created earlier.

Set these environment variables to appropriate values:

```
export INFRA_ACCESS_KEY="<connector access key>"
export DESTINATION_HOST="<public ip or hostname>"
export DESTINATION_NAME=example
```

And create the connector configuration file using those variables:

```
cat << EOF | sudo tee /etc/infra/connector.yaml
kind: ssh
name: $DESTINATION_NAME
endpointAddr: $DESTINATION_HOST
server:
  accessKey: $INFRA_ACCESS_KEY
EOF
sudo chmod 600 /etc/infra/connector.yaml
sudo chown infra:infra /etc/infra/connector.yaml
```

> Currently, the name of the destination (`example` in the configuration above) cannot contain dots, and therefore can not be the IP address of the SSH host.

Configure `sshd` to use Infra:

```
cat << EOF | sudo tee -a /etc/ssh/sshd_config
Match group infra-users
  AuthorizedKeysFile none
  PasswordAuthentication no
  AuthorizedKeysCommand /usr/local/sbin/infra sshd auth-keys %u %f
  AuthorizedKeysCommandUser infra
EOF
```

Finally, restart `sshd` service and start `infra`:

```
sudo systemctl restart sshd
sudo systemctl restart infra
sudo systemctl enable infra
```

Your SSH host should be ready to receive connections!

## Connect

On your desktop machine, give yourself access:

```
infra grants add <your email> example
```

`example` is the name of the destination from `connector.yaml` above.

> Granting access to groups is not currently supported.

Use `infra list` to see you have access through Infra.

Next, access the server:

```bash
ssh <destination ip address>
```

You should be automatically authenticated and logged in. See your username:

```
whoami
```

## Access Control

SSH access control is binary: either a user has access, or they don't.

For example, to grant a user access to a server:

```bash
infra grants add suzie@infrahq.com example
```

To revoke access:

```bash
infra grants remove suzie@infrahq.com example
```

## Customizing

### Sudo access

Setup `sudo` to allow your user access to other accounts on the machine.

```
USERNAME=suzie
echo "$USERNAME ALL=(ALL) NOPASSWD:ALL" | sudo tee /etc/sudoers.d/$USERNAME
```

### User provisioning

Infra creates users on the SSH host with `useradd`. The default shell of the user, and
other settings for `useradd` can be customzied with `/etc/defaults/useradd`. See the
[`useradd` man page](https://manpages.ubuntu.com/manpages/xenial/en/man8/useradd.8.html) for more details.

When a user's access is removed the user and their home directory will be removed from the system
with `userdel`.
