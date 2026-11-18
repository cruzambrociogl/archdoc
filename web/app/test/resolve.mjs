// The resolve hook register.mjs installs: a relative import with no extension tries .ts.
export async function resolve(specifier, context, next) {
  if (/^\.\.?\//.test(specifier) && !/\.[cm]?[jt]sx?$/.test(specifier)) {
    try {
      return await next(specifier + '.ts', context)
    } catch {
      // Not a .ts file: let Node say what it is.
    }
  }
  return next(specifier, context)
}
