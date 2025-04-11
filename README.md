arptables iptables ebtables required

## Download cloud hypervisor
```bash
wget https://github.com/cloud-hypervisor/cloud-hypervisor/releases/download/v44.0/cloud-hypervisor -O ./resources/cloud-hypervisor
chmod +x ./resources/cloud-hypervisor
wget https://github.com/cloud-hypervisor/linux/releases/download/ch-release-v6.12.8-20250114/vmlinux -O ./resources/vmlinux
```

## Building the kernel

```bash
wget https://github.com/cloud-hypervisor/linux/releases/download/ch-release-v6.12.8-20250114/vmlinux -O ./resources/vmlinux
```


pvcreate /dev/nvme0n1p4
vgcreate vg_baepo /dev/nvme0n1p4
lvcreate -L 180G -T vg_baepo/thinpool



CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o resources/baepo-initd -ldflags "-s -w" cmd/baepo-initd/main.go
