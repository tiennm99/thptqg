// Every route is a static file: no server runs, and the databases are fetched
// from the browser. Trailing slashes make each page a directory index, which is
// what lets GitHub Pages serve /thptqg/2016/ without a rewrite.
export const prerender = true;
export const trailingSlash = "always";
