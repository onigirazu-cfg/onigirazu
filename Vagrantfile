Vagrant.configure("2") do |config|
  config.vm.box_check_update = false
  config.vm.synced_folder ".", "/vagrant", disabled: false

  config.ssh.insert_key = false
  config.ssh.private_key_path = ["~/.vagrant.d/insecure_private_key", "~/.ssh/id_rsa"]

  config.vm.provider "virtualbox" do |vb|
    vb.memory = "512"
    vb.cpus = 1
    vb.linked_clone = true
  end

  machines = {
    "ubuntu2004" => {
      box: "bento/ubuntu-20.04-arm64",
      ip: "192.168.56.10",
      hostname: "ubuntu2004.test"
    },
    "ubuntu2204" => {
      box: "ubuntu/jammy64",
      ip: "192.168.56.11",
      hostname: "ubuntu2204.test"
    },
    "ubuntu2404" => {
      box: "bento/ubuntu-24.04",
      ip: "192.168.56.12",
      hostname: "ubuntu2404.test"
    },
    "debian11" => {
      box: "debian/bullseye64",
      ip: "192.168.56.20",
      hostname: "debian11.test"
    },
    "debian12" => {
      box: "debian/bookworm64",
      ip: "192.168.56.21",
      hostname: "debian12.test"
    },
    "centos7" => {
      box: "centos/7",
      ip: "192.168.56.30",
      hostname: "centos7.test"
    },
    "rocky8" => {
      box: "generic/rocky8",
      ip: "192.168.56.31",
      hostname: "rocky8.test"
    },
    "rocky9" => {
      box: "generic/rocky9",
      ip: "192.168.56.32",
      hostname: "rocky9.test"
    },
    "opensuse15" => {
      box: "opensuse/Leap-15.5.x86_64",
      ip: "192.168.56.40",
      hostname: "opensuse15.test"
    },
    "freebsd13" => {
      box: "generic/freebsd13",
      ip: "192.168.56.50",
      hostname: "freebsd13.test"
    },
    "freebsd14" => {
      box: "generic/freebsd14",
      ip: "192.168.56.51",
      hostname: "freebsd14.test"
    }
  }

  machines.each do |name, machine|
    config.vm.define name, autostart: false do |node|
      node.vm.box = machine[:box]
      node.vm.hostname = machine[:hostname]
      node.vm.network "private_network", ip: machine[:ip]

      node.vm.provision "shell", inline: <<-SHELL
        if command -v apt-get &> /dev/null; then
          apt-get update -qq
          apt-get install -y -qq python3 sudo
        elif command -v yum &> /dev/null; then
          yum install -y -q python3 sudo
        elif command -v zypper &> /dev/null; then
          zypper install -y python3 sudo
        elif command -v pkg &> /dev/null; then
          pkg install -y python3 sudo
        fi

        if ! id -u testuser &> /dev/null; then
          if command -v useradd &> /dev/null; then
            useradd -m -s /bin/bash testuser
          else
            pw useradd testuser -m -s /bin/sh
          fi
          echo "testuser:testpass" | chpasswd 2>/dev/null || echo "testpass" | pw usermod testuser -h 0

          if [ -d /etc/sudoers.d ]; then
            echo "testuser ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/testuser
            chmod 0440 /etc/sudoers.d/testuser
          fi
        fi

        mkdir -p /home/testuser/.ssh
        cat /home/vagrant/.ssh/authorized_keys > /home/testuser/.ssh/authorized_keys 2>/dev/null || true
        chown -R testuser:testuser /home/testuser/.ssh 2>/dev/null || chown -R testuser:testuser /home/testuser/.ssh
        chmod 700 /home/testuser/.ssh
        chmod 600 /home/testuser/.ssh/authorized_keys
      SHELL
    end
  end
end
