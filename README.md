# JenkinsCheck

This is a little app to monitor jenkins jobs and get notified, if a job is failing or getting
unstable. It is much like [CCTray](https://sourceforge.net/projects/ccnet/) but it also handles unstable build status.

The cc.xml standard does not have a unstable status. Unstable build results show up as failure, but
via Jenkins' XML API it can be queried if a build result was really a failure or just unstable.
