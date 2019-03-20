# utilities
Reuseable Go utilities used by other projects

MUST DO: 
If you make changes to this service you must update the template and any micro-services that requrie your changes to use the latest version of this package.
You do this by removing the folder 'vendor/github.impcloud.net/Responsive-Retail-Core/utilities/' from all micro-services and run 'govendor fetch -tree github.impcloud.net/Responsive-Retail-Core/utilities/^::github.impcloud.net/Responsive-Retail-Core/utilities.git' on every service.

Clone to your GOPATH/src/github.impcloud.net/Responsive-Retail-Core folder. This is a must so import statements are correct and consistent across RRP projects.

We will be using GoVendor to manage dependecies. GoVendor manages the projects dependencies locally in the vendor folder and tracks them via the vendor/vendor.json file.

Use: go get -u github.com/kardianos/govendor to install GoVendor.

Run 'govendor sync' from the project base folder to pull all missing dependencies.

Run 'govendor fetch [package]' from the project base folder to pull new dependencies, store them in the local vendor folder and added version details to the vendor/vendor.json. - This is instead of running 'go get [package]', which we will no longer use.

