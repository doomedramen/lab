# Vagrantfile for x86_64 emulation on Apple Silicon (M1/M2/M3)
# Based on: https://medium.com/@lijia1/x86-64-emulation-on-apple-silicon-1086639f6dfc
#
# Prerequisites:
#   brew install --cask vagrant
#   brew install qemu
#   vagrant plugin install vagrant-qemu
#
# Usage:
#   vagrant up              # Start VM and provision
#   vagrant ssh             # SSH into VM
#   vagrant ssh -c "cd /vagrant && go vet ./..."  # Run commands
#   vagrant reload --provision  # Re-provision
#   vagrant halt            # Stop VM
#   vagrant destroy         # Delete VM

Vagrant.configure("2") do |config|
  # Use Ubuntu 22.04 (x86_64)
  config.vm.box = "generic/ubuntu2204"
  
  # QEMU provider configuration
  config.vm.provider "qemu" do |qe|
    # Emulate x86_64 architecture
    qe.arch = "x86_64"
    qe.machine = "q35"
    qe.cpu = "Skylake-Client-v1"
    qe.net_device = "virtio-net-pci"
    
    # Allocate resources (adjust based on your Mac)
    # M1/M2/M3: Can allocate more than physical cores via emulation
    qe.smp = "cpus=4,sockets=1,cores=4,threads=1"
    qe.memory = "8192"
    
    # Port forwarding (optional)
    qe.extra_netdev_args = "hostfwd=tcp::8080-:8080"
  end
  
  # Mount project folder using rsync (more reliable than SMB on macOS)
  config.vm.synced_folder ".", "/vagrant", type: "rsync",
    rsync__exclude: [
      ".git/",
      "node_modules/",
      ".pnpm-store/",
      "apps/*/node_modules/",
      "apps/web/.next/",
      "apps/web/out/",
      "apps/api/bin/",
      "apps/api/gen/",
      "apps/web/lib/gen/",
      ".dockerignore",
      "*.md"
    ],
    rsync__auto: false
  
  # Provisioning script - installs all dependencies
  config.vm.provision "shell", inline: <<-SHELL
    set -e
    
    echo "=== Updating system packages ==="
    apt-get update
    
    echo "=== Installing Go ==="
    GOVERSION="1.25.7"
    wget -q "https://go.dev/dl/go${GOVERSION}.linux-amd64.tar.gz"
    rm -rf /usr/local/go
    tar -C /usr/local -xzf "go${GOVERSION}.linux-amd64.tar.gz"
    rm "go${GOVERSION}.linux-amd64.tar.gz"
    echo 'export PATH="/usr/local/go/bin:$PATH"' >> /home/vagrant/.bashrc
    
    echo "=== Installing libvirt and build tools ==="
    apt-get install -y \
      libvirt-dev \
      pkg-config \
      gcc \
      libc6-dev \
      make \
      git \
      curl \
      wget
    
    echo "=== Installing buf for proto generation ==="
    curl -sSL "https://github.com/bufbuild/buf/releases/latest/download/buf-$(uname -s)-$(uname -m)" -o /usr/local/bin/buf
    chmod +x /usr/local/bin/buf
    
    echo "=== Setting up Go workspace ==="
    mkdir -p /home/vagrant/go/bin
    chown -R vagrant:vagrant /home/vagrant/go
    
    echo "=== Installing Go tools ==="
    su - vagrant -c "export PATH=/usr/local/go/bin:\$PATH && go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"
    su - vagrant -c "export PATH=/usr/local/go/bin:\$PATH && go install connectrpc.com/connect/cmd/protoc-gen-connect-go@latest"
    
    echo "=== Generating proto files ==="
    su - vagrant -c "export PATH=/usr/local/go/bin:/home/vagrant/go/bin:\$PATH && cd /vagrant/packages/proto && buf generate --template buf.gen.api.yaml"
    
    echo "=== Running go vet ==="
    su - vagrant -c "export PATH=/usr/local/go/bin:/home/vagrant/go/bin:\$PATH && cd /vagrant/apps/api && go vet ./..."
    
    echo "=== Running go tests ==="
    su - vagrant -c "export PATH=/usr/local/go/bin:/home/vagrant/go/bin:\$PATH && export JWT_SECRET='test-secret-at-least-32-characters-long' && cd /vagrant/apps/api && go test -v -short ./..."
    
    echo "=== Provisioning complete ==="
  SHELL
end
