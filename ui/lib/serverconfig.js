import useSWR from 'swr'

export function useServerConfig() {
  const { data: { isEmailConfigured, isSignupEnabled, baseDomain } = {} } =
    useSWR(`/api/server-configuration`, {
      revalidateIfStale: false
    })

  return {
    isEmailConfigured,
    isSignupEnabled,
    baseDomain,
  }
}
