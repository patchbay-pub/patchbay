This is a CLI companion tool for the patchbay.pub web service.

It is still very early in development. See main.go for all functionality and
uses.

To host a static site, run something like the following from the root directory
of the site:

```
patchbay-cli host -root-channel https://patchbay.pub/my-secret-channel
```

You can then navigate to `https://patchbay.pub/my-secret-channel/` in your
browser. Don't forget the trailing slash, or use the full /index.html URL.
This is necessary for relative URLs to work properly in browsers.
