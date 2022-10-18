import { useRouter } from 'next/router'
import { useState } from 'react'

import DashboardLayout from '../../components/layouts/dashboard'

export default function DeviceShow() {
  const router = useRouter()
  const [code, setCode] = useState(router.query.code)
  const [codeEntered, setCodeEntered] = useState(false)
  const [error, setError] = useState('')
  
  async function onSubmit(e) {
    e.preventDefault()

    if (code.length == 8)
      setCode(code.substring(0, 4) + '-' + code.substring(4,8))

    try {
      const res = await fetch('/api/device/approve', {
        method: 'post',
        body: JSON.stringify({
          userCode: code,
        }),
      })

      await jsonBody(res)

      setCodeEntered(true)
    } catch (e) {
      setError(e.message)
    }

    return false
  }

  return (
    <div className='flex min-h-[280px] w-full flex-col items-center px-10 py-8'>
     <>
          <h1 className='text-base font-bold leading-snug'>Device Code</h1>
          {codeEntered ? (
            <p className='my-3 flex max-w-[260px] flex-1 flex-col items-center justify-center text-center text-s text-gray-600'>
              You've authorized the new device. You can close this window.
            </p>
          ) : (
            <>
              <h2 className='my-1.5 mb-4 max-w-md text-center text-xs text-gray-500'>
                This code gives a new device access to your account. Never accept a device code from someone else.
              </h2>
              <form
                onSubmit={onSubmit}
                className='relative flex w-full max-w-sm flex-1 flex-col justify-center'
              >
                <div className='my-2 w-full'>
                  <label
                    htmlFor='code'
                    className='text-2xs font-medium text-gray-700'
                  >
                    Device Code
                  </label>
                  <input
                    required
                    autoFocus
                    placeholder='AAAA-BBBB'
                    type='text'
                    name='code'
                    style={{"textTransform": 'uppercase'}}
                    value={code}
                    onChange={e => setCode(e.target.value)}
                    className={`mt-1 block w-full rounded-md shadow-sm focus:border-blue-500 focus:ring-blue-500 lg:text-lg ${
                      error ? 'border-red-500' : 'border-gray-300'
                    }`}
                  />
                  {error && (
                    <p className='absolute top-full mt-1 text-xs text-red-500'>
                      {error}
                    </p>
                  )}
                </div>
                <button
                  disabled={codeEntered}
                  className='mt-4 mb-2 flex w-full cursor-pointer justify-center rounded-md border border-transparent bg-blue-500 py-2 px-4 font-medium text-white shadow-sm hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 sm:text-sm'
                >
                  Confirm and Authorize New Device
                </button>
              </form>
            </>
          )}
        </>
    </div>
  )
}

DeviceShow.layout = page => <DashboardLayout>{page}</DashboardLayout>
