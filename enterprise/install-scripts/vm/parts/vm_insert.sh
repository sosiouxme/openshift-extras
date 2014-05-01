# do the things that are specific to the VM
setup_vm()
{
  setup_vm_host
  setup_vm_user
  clean_vm
}

# The RHEL + OSE VM requires the setup of a default user.
setup_vm_host()
{
  # create a hook that updates the DNS record when our IP changes
  local name=${broker_hostname%$hosts_domain}
  cat <<HOOK > /etc/dhcp/dhclient-eth0-up-hooks
    if [[ "\$new_ip_address"x != x ]]; then
      /usr/sbin/rndc freeze ${hosts_domain}
      sed -i -e "s/^vm\s\+\(IN \)\?A\s\+.*/vm A \$new_ip_address/" /var/named/dynamic/${hosts_domain}.db
      /usr/sbin/rndc thaw ${hosts_domain}
      sed -i -e "s/^PUBLIC_IP=.*/PUBLIC_IP=\$new_ip_address/" /etc/openshift/node.conf
    fi
HOOK
  chmod +x /etc/dhcp/dhclient-eth0-up-hooks

  # modify selinux policy to allow this script to change named from dhcp client
  cat <<POLICY > /tmp/dhcp-update-named.te
module dhcp-update-named 1.0;

require {
        type etc_t;
        type dnssec_t;
        type ndc_exec_t;
        type named_zone_t;
        type named_cache_t;
        type dhcpc_t;
        class dir { write remove_name search add_name };
        class file { rename execute setattr read create ioctl execute_no_trans write getattr unlink open };
}

# allow to edit node.conf
allow dhcpc_t etc_t:file { write rename create unlink setattr };
# allow to read rndc key
allow dhcpc_t dnssec_t:file { read getattr open };
# allow to run rndc
allow dhcpc_t ndc_exec_t:file { read getattr open execute execute_no_trans };
# allow to descend into /var/named
allow dhcpc_t named_zone_t:dir search;
# allow to change /var/named/dynamic/*.db
allow dhcpc_t named_cache_t:dir { write remove_name search add_name };
allow dhcpc_t named_cache_t:file { rename setattr read create ioctl write getattr unlink open };

POLICY
  pushd /tmp
    make -f /usr/share/selinux/devel/Makefile
    semodule -i dhcp-update-named.pp
    rm dhcp-update-named.*
  popd

  # Set the runlevel to graphical
  /bin/sed -i -e 's/id:.:initdefault:/id:5:initdefault:/' /etc/inittab

  # prevent warnings about certificate, at least on the host
  cat <<INSECURE >> /etc/openshift/express.conf
# Ignore certificate errors. VM is installed with self-signed certificate.
insecure=true
INSECURE

  # accept the server certificate in Firefox
  #certFile='/etc/pki/tls/certs/localhost.crt'
  #certName='OpenShift Enterprise VM'
  #certutil -A -n "${certName}" -t "TCu,Cuw,Tuw" -i ${certFile} -d /etc/pki/nssdb/

  # no need for root to login with a password.
  /usr/bin/passwd -l root
}

setup_vm_user()
{
  # Create the 'openshift' user
  /usr/sbin/useradd openshift
  /bin/echo 'openshift:openshift' | /usr/sbin/chpasswd -c SHA512
  # Set up the 'openshift' user for auto-login
  /usr/sbin/groupadd nopasswdlogin
  /usr/sbin/usermod -G openshift,nopasswdlogin openshift
  /bin/sed -i -e '
/^\[daemon\]/a \
AutomaticLogin=openshift \
AutomaticLoginEnable=true
/^\[greeter\]/a \
IncludeAll=false \
Include=openshift
' /etc/gdm/custom.conf
  /bin/sed -i -e '1i \
auth sufficient pam_succeed_if.so user ingroup nopasswdlogin' /etc/pam.d/gdm-password
  # add the user to sudo
  echo "openshift ALL=(ALL)  NOPASSWD: ALL" > /etc/sudoers.d/openshift
  # Disable locking the user desktop for inactivity
  su - openshift -c 'gconftool-2 -s /apps/gnome-screensaver/idle_activation_enabled --type=bool false'
  # TODO: automatically log the user in
  # TODO: get rid of email icon, add terminal icon

  # fabricated function to lay down files from the parts/ dir
  create_user_files

  # accept the server certificate in Firefox
  # TODO: this does nothing until Firefox has been run to create a profile
  certFile='/etc/pki/tls/certs/localhost.crt'
  certName='OpenShift Enterprise VM'
  for db in $(find  /home/openshift/.mozilla* -name "cert8.db"); do
    certutil -A -n "${certName}" -t "TCu,Cuw,Tuw" -i ${certFile} -d "$(dirname ${db})"
  done

  # install oo-install and default config
  wget $OO_INSTALL_URL -O /home/openshift/oo-install.zip --no-check-certificate -nv
  su - openshift -c 'unzip oo-install.zip -d oo-install'
  rm /home/openshift/oo-install.zip 

  # fix ownership
  chown -R openshift:openshift /home/openshift

  # install JBoss Developer Suite
  wget $JBDS_URL -O /home/openshift/jbds.jar --no-check-certificate -nv
  # https://access.redhat.com/site/solutions/44667 for auto install
  su - openshift -c 'java -jar jbds.jar jbdevstudio/jbds-install.xml' && rm /home/openshift/jbds.jar
}

clean_vm()
{
  # clean vm of anything it should not keep
  if [ "$DEBUG_VM"x = "x" ]; then
    # items to skip when debugging:
    rm -f /etc/yum.repos.d/*
    yum clean all
    rm -f /root/anaconda*
    #virt-sysprep --enable abrt-data,bash-history,dhcp-client-state,machine-id,mail-spool,pacct-log,smolt-uuid,ssh-hostkeys,sssd-db-log,udev-persistent-net,utmp,net-hwaddr
  fi
  # clean even when debugging
  rm /etc/udev/rules.d/70-persistent-net.rules  # keep specific NIC from being recorded
  sed -i -e '/^HWADDR/ d' /etc/sysconfig/network-scripts/ifcfg-eth0 # keep HWADDR from being recorded
}

