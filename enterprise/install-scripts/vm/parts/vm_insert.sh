
# The RHEL + OSE VM requires the setup of a default user.
setup_vm_user()
{
  # Set the runlevel to graphical
  /bin/sed -i -e 's/id:.:initdefault:/id:5:initdefault:/' /etc/inittab

  # Create the 'openshift' user
  /usr/sbin/useradd openshift

  # Set the account password
  /bin/echo 'openshift:openshift' | /usr/sbin/chpasswd -c SHA512

  # Set up the 'openshift' user for auto-login
  /usr/sbin/groupadd nopasswdlogin
  /usr/sbin/usermod -G openshift,nopasswdlogin openshift
  /bin/sed -i -e '/^\[daemon\]/a \
AutomaticLogin=openshift \
AutomaticLoginEnable=true \
' /etc/gdm/custom.conf
  /bin/sed -i -e '1i \
auth sufficient pam_succeed_if.so user ingroup nopasswdlogin' /etc/pam.d/gdm-password

  # Place a "Welcome to OpenShift" page in the user homedir
  mkdir -p /home/openshift/.openshift/
  create_welcome_file

  # Place a startup routine in the user homedir
  mkdir -p /home/openshift/.config/autostart/
  create_desktop_file
}

clean_vm()
{
  if [ "$DEBUG_VM"x = "x" ]; then
    rm -f /etc/yum.repos.d/*
    yum clean all
    rm -f /root/anaconda*
  fi

  # items to do even when debugging:
  rm /etc/udev/rules.d/70-persistent-net.rules  # this keeps the NIC from being recorded
}

