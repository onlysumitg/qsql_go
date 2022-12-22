internal is a special go dir. This can not be imported outside of its parent folder "QSQL2" in this case(lets go page 40)

=>   a package /a/b/c/internal/d/e/f can only be imported by code in the directory tree rooted at /a/b/c. 
===> It cannot be imported by code in /a/b/g or in any other repository. 

=> this should contain non app specific code which can be used by multiple apps
    => like models --> can be used by web app as well as cli app



