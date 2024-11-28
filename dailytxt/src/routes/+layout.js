export const ssr = false;
//export const prerender = false;

/**
 * @param {{ url: URL }} param0
 */
export const load = ({ url }) => {
  const { pathname } = url

  return {
    pathname
  }
}