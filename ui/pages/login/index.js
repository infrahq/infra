import { useRouter } from 'next/router'
import { useState } from 'react'
import useSWR, { useSWRConfig } from 'swr'

import { kind } from '../../lib/providers'
import LoginLayout from '../../components/layouts/login'

function oidcLogin ({ id, url, clientID }) {
  window.localStorage.setItem('providerID', id)

  const state = [...Array(10)].map(() => (~~(Math.random() * 36)).toString(36)).join('')
  window.localStorage.setItem('state', state)

  const redirectURL = window.location.origin + '/login/callback'
  window.localStorage.setItem('redirectURL', redirectURL)

  document.location.href = `https://${url}/oauth2/v1/authorize?redirect_uri=${redirectURL}&client_id=${clientID}&response_type=code&scope=openid+email+groups+offline_access&state=${state}`
}

function Providers ({ providers }) {
  return (
    <>
      <div className='w-full max-w-sm mt-2'>
        {providers?.map(p => (
          <button onClick={() => oidcLogin(p)} key={p.id} className='w-full border border-gray-800 hover:to-pink-50 rounded-md p-0.5 my-1.5'>
            <div className='flex flex-col items-center justify-center px-4 py-2'>
              {kind(p.url)
                ? (
                  <div className='flex flex-col items-center text-center py-0.5'>
                    <img className='h-4' src={`/providers/${kind(p.url)}.svg`} />
                    {providers?.length > 1 && <div className='text-2xs text-gray-300'>{p.url}</div>}
                  </div>
                  )
                : <p className='font-bold h-4 m-1'>Single Sign-On</p>}
            </div>
          </button>
        ))}
      </div>
      <div className='w-full mt-4 relative'>
        <div className='absolute inset-0 flex items-center' aria-hidden='true'>
          <div className='w-full border-t border-gray-800' />
        </div>
        <div className='relative flex justify-center text-sm'>
          <span className='px-2 bg-black text-2xs text-gray-300'>OR</span>
        </div>
      </div>
    </>
  )
}

export default function Login () {
  const { data: { items: providers } = {} } = useSWR('/api/providers', { fallbackData: [] })
  const { mutate } = useSWRConfig()
  const router = useRouter()

  const [name, setName] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')

  async function onSubmit (e) {
    e.preventDefault()

    try {
      const res = await fetch('/api/login', {
        method: 'post',
        body: JSON.stringify({
          passwordCredentials: {
            name,
            password
          }
        })
      })

      if (!res.ok) {
        throw await res.json()
      }

      const data = await res.json()

      if (data.passwordUpdateRequired) {
        router.replace({
          pathname: '/login/finish',
          query: { id: data.polymorphicID.replace('i:', '') }
        })
        return
      }

      mutate('/api/users/self', { optimisticData: { name } })

      router.replace('/')
    } catch (e) {
      console.error(e)
      setError('Invalid credentials')
    }

    return false
  }

  return (
    <>
      <h1 className='text-base leading-snug font-bold'>Login to Infra</h1>
      <h2 className='text-xs text-center max-w-[260px] my-3 text-gray-300'>Welcome back. Login with your credentials {providers.length > 0 && 'or via your identity provider.'}</h2>
      {providers?.length > 0 && <Providers providers={providers} />}

      <form onSubmit={onSubmit} className='flex flex-col w-full max-w-sm relative'>
        <div className='w-full my-2'>
          <label htmlFor='name' className='text-3xs text-gray-500 uppercase'>Username</label>
          <input
            required
            autoFocus
            name='name'
            placeholder='enter your username or email'
            onChange={e => {
              setName(e.target.value)
              setError('')
            }}
            className={`w-full bg-transparent border-b border-gray-800 text-2xs px-px py-2 focus:outline-none focus:border-b focus:border-gray-200 placeholder:italic ${error ? 'border-pink-500/60' : ''}`}
          />
        </div>
        <div className='w-full my-2'>
          <label htmlFor='password' className='text-3xs text-gray-500 uppercase'>Password</label>
          <input
            required
            name='password'
            type='password'
            placeholder='enter your password'
            onChange={e => {
              setPassword(e.target.value)
              setError('')
            }}
            className={`w-full bg-transparent border-b border-gray-800 text-2xs px-px py-2 focus:outline-none focus:border-b focus:ring-gray-200 placeholder:italic ${error ? 'border-pink-500/60' : ''}`}
          />
        </div>
        <button disabled={!name || !password} className='border border-violet-300 hover:border-violet-100 mt-6 mb-2 text-2xs px-4 py-3 rounded-lg disabled:pointer-events-none text-violet-100 disabled:opacity-30'>
          Login
        </button>
        {error && <p className='absolute -bottom-3.5 w-full mx-auto text-2xs text-pink-400 text-center'>{error}</p>}
      </form>
    </>
  )
}

Login.layout = page => <LoginLayout>{page}</LoginLayout>
