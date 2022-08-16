import useSWR from 'swr'

export function serverConfig() {
  const { data: { isEmailConfigured, isSignupEnabled } = {} } = useSWR(
    `/api/server-config`,
    {
      revalidateIfStale: false,
    }
  )

  return {
    isEmailConfigured,
    isSignupEnabled,
  }
}
