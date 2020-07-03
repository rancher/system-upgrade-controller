# Rolling OS upgrades

This folder containsthe plans for:

-  Ubuntu 18.04 ([`bionic.yaml`](bionic.yaml)) 
-  Centos 7 ([`centos7.yaml`](centos7.yaml)). 

These plans will upgrade all parts of the OS, and will do a install of docker after reboot.

The [`upgrade.sh`](upgrade.sh) file will install the plans, label all nodes and force an upgrade of the nodes.
