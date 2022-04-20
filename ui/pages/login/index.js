import { useRouter } from 'next/router'
import { useState } from 'react'
import useSWR, { useSWRConfig } from 'swr'

function oidcLogin ({ id, url, clientID }) {
  window.localStorage.setItem('providerId', id)

  const state = [...Array(10)].map(() => (~~(Math.random() * 36)).toString(36)).join('')
  window.localStorage.setItem('state', state)

  const redirectURL = window.location.origin + '/login/callback'
  window.localStorage.setItem('redirectURL', redirectURL)

  document.location.href = `https://${url}/oauth2/v1/authorize?redirect_uri=${redirectURL}&client_id=${clientID}&response_type=code&scope=openid+email+groups+offline_access&state=${state}`
}

export default function () {
  const { data: providers } = useSWR('/v1/providers', { fallbackData: [] })
  const { mutate } = useSWRConfig()
  
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const router = useRouter()

  async function login () {
    try {
      const res = await fetch('/v1/login', { method: 'post', body: JSON.stringify({ passwordCredentials: { email, password } }) })
      if (res.data.passwordUpdateRequired) {
        router.replace({
          pathname: '/login/finish',
          query: { id: res.data.polymorphicId.split(':')[1] }
        })
        return
      }

      await mutate('/v1/introspect')
      router.replace('/')
    } catch (e) {
      console.log(e)
      setError('Invalid credentials')
    }
  }

  return (
    <div className='flex flex-col justify-center items-center h-full w-full max-w-sm mx-auto mb-48'>
      <img className='text-white w-10 h-10' src='/infra-icon.svg' />
      <h1 className='my-5 text-3xl font-light tracking-tight'>Login to Infra</h1>

      {providers.length > 0 && (
        <>
          <div className='w-full mt-8'>
            {providers.map(p => (
              <button
                onClick={() => oidcLogin(p)}
                key={p.id}
                className='w-full flex items-center justify-center border border-gray-600 hover:border-gray-500 py-3.5 rounded-md'
              >
                <img className='h-4' src='/providers/okta.svg' />
              </button>
            ))}
          </div>
          <div className='w-full my-6 relative'>
            <div className='absolute inset-0 flex items-center' aria-hidden='true'>
              <div className='w-full border-t border-gray-800' />
            </div>
            <div className='relative flex justify-center text-sm'>
              <span className='px-8 bg-black text-gray-500'>or login with</span>
            </div>
          </div>
        </>
      )}

      {/* login form */}
      <form
        className='w-full flex flex-col max-w-sm relative'
        onSubmit={e => {
          e.preventDefault()
          login()
        }}
      >
        <input
          required
          type='text'
          name='name'
          id='name'
          className={`block w-full px-4 py-2 text-md border font-light rounded-t-lg text-zinc-100 bg-zinc-900/50 placeholder-gray-500 ${error ? 'border-red-500' : 'border-zinc-800'}`}
          placeholder='email'
          onChange={e => {
            setError('')
            setEmail(e.target.value)
          }}
        />
        <input
          required
          type='password'
          name='password'
          id='password'
          className={`block w-full px-4 py-2 text-md -my-px border font-light rounded-b-lg text-zinc-100 bg-zinc-900/50 placeholder-gray-500 ${error ? 'border-red-500' : 'border-zinc-800'}`}
          placeholder='password'
          onChange={e => {
            setError('')
            setPassword(e.target.value)
          }}
        />
        <input type='submit' value='Login' className='w-full my-3 bg-zinc-500/20 hover:bg-gray-500/25 py-2.5 rounded-md text-white text-md hover:cursor-pointer' />
        {error && (
          <p className='mt-2 text-sm absolute text-red-500 -bottom-4'>
            {error}
          </p>
        )}
      </form>
    </div>
  )
}
