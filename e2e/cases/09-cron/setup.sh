# a line onigirazu does not manage: it must survive the run
( crontab -l -u root 2>/dev/null | grep -v 'onigirazu-e2e-unmanaged' ; echo '5 4 * * * /bin/echo onigirazu-e2e-unmanaged' ) | crontab -u root -
