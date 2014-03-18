configure_repos()
{
  echo "OpenShift: Begin configuring repos."
  cat > /etc/yum.repos.d/openshift.repo <<YUM

[openshift]
name=OpenShift Enterprise
baseurl=file:///mnt/redhat/brewroot/repos/rhose-2.0-rhel-6-build/latest/x86_64
enabled=1
gpgcheck=0
priority=10

[rhscl-ruby193]
name=RHSCL 1.0 ruby193
baseurl=file:///mnt/redhat/brewroot/repos/ruby193-rhel-6-build/latest/x86_64
enabled=1
gpgcheck=0
priority=10

[rhscl-nodejs010]
name=RHSCL 1.0 nodejs010
baseurl=file:///mnt/redhat/brewroot/repos/nodejs010-rhel-6-build/latest/x86_64
enabled=1
gpgcheck=0
priority=10

[rhscl-python27]
name=RHSCL 1.0 python27
baseurl=file:///mnt/redhat/brewroot/repos/python27-rhel-6-build/latest/x86_64
enabled=1
gpgcheck=0
priority=10

[rhscl-postgresql92]
name=RHSCL 1.0 postgresql92
baseurl=file:///mnt/redhat/brewroot/repos/postgresql92-rhel-6-build/latest/x86_64
enabled=1
gpgcheck=0
priority=10

[rhel6]
name=RHEL 6 base OS
baseurl=file:///mnt/redhat/rel-eng/repos/RHEL-6-block-tomcat/x86_64
enabled=1
gpgcheck=0
priority=20

[jbosseap6]
name=JBoss EAP
baseurl=file:///mnt/redhat/released/JBEAP-6/x86_64
enabled=1
gpgcheck=0
priority=30
sslverify=false

[jbossews2]
name=JBoss EWS 2
baseurl=file:///mnt/redhat/rel-eng/repos/jb-ews-2-rhel-6-current/x86_64
enabled=1
gpgcheck=0
priority=30
sslverify=false

YUM
  echo "OpenShift: Completed configuring repos."
}

clear_repos()
{
  rm -f /etc/yum.repos.d/openshift.repo
}

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
  # TODO: Create a symlink to the RPM-installed location of welcome.html
  # ln -s path/to/welcome.html /home/openshift/.openshift/welcome.html

  # Place a startup routine in the user homedir
  mkdir -p /home/openshift/.config/autostart/
  # TODO: Create a symlink to the RPM-installed location of com.redhat.OSEWelcome.desktop
  # ln -s path/to/com.redhat.OSEwelcome.desktop /home/openshift/.config/autostart/com.redhat.OSEwelcome.desktop
}

