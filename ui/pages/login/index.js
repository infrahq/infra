import { useRouter } from 'next/router'
import { useState } from 'react'
import useSWR, { useSWRConfig } from 'swr'

import { kind } from '../../lib/providers'

import HeaderIcon from '../../components/header-icon'

function oidcLogin ({ id, url, clientID }) {
  window.localStorage.setItem('providerId', id)

  const state = [...Array(10)].map(() => (~~(Math.random() * 36)).toString(36)).join('')
  window.localStorage.setItem('state', state)

  const redirectURL = window.location.origin + '/login/callback'
  window.localStorage.setItem('redirectURL', redirectURL)

  document.location.href = `https://${url}/oauth2/v1/authorize?redirect_uri=${redirectURL}&client_id=${clientID}&response_type=code&scope=openid+email+groups+offline_access&state=${state}`
}

function Providers ({ providers }) {
  return (
    <>
      <div className='w-full max-w-sm mt-8'>
        {providers?.map(p => (
          <button onClick={() => oidcLogin(p)} key={p.id} className='w-full border border-gray-800 hover:to-pink-50 rounded-md p-0.5 my-1.5'>
            <div className='flex flex-col items-center justify-center px-4 py-2'>
              {kind(p.url)
                ? (
                  <button className='flex flex-col items-center text-center py-0.5'>
                    <img className='h-4' src={`/providers/${kind(p.url)}.svg`} />
                    {providers?.length > 1 && <div className='text-xs text-gray-300'>{p.url}</div>}
                  </button>
                  )
                : <p className='font-bold h-4 m-1'>Single Sign-On</p>}
            </div>
          </button>
        ))}
      </div>
      <div className='w-full my-8 relative'>
        <div className='absolute inset-0 flex items-center' aria-hidden='true'>
          <div className='w-full border-t border-gray-800' />
        </div>
        <div className='relative flex justify-center text-sm'>
          <span className='px-2 bg-black text-xs text-gray-300'>OR</span>
        </div>
      </div>
    </>
  )
}

export default function () {
  const { data: providers } = useSWR('/v1/providers', { fallbackData: [] })
  const { mutate } = useSWRConfig()
  const router = useRouter()

  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')

  async function onSubmit (e) {
    e.preventDefault()

    try {
      const res = await fetch('/v1/login', {
        method: 'post',
        body: JSON.stringify({
          passwordCredentials: {
            email,
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

      mutate('/v1/identities/self', { optimisticData: { name: email } })

      router.replace('/')
    } catch (e) {
      console.error(e)
      setError('Invalid credentials')
    }

    return false
  }

  return (
    <div className='w-full min-h-full flex flex-col'>
      <div className='flex flex-col justify-center items-center px-5 py-5 my-10 border rounded-lg border-gray-800'>
        <HeaderIcon size={12} iconPath='/infra-color.svg' />
        <h1 className='text-base leading-snug font-bold'>Login to Infra</h1>
        <h2 className='text-[13px] text-center max-w-[260px] my-3 text-gray-300'>Welcome back. Login with your credentials {providers.length > 0 && 'or via your identity provider.'}</h2>
        {providers?.length > 0 && <Providers providers={providers} />}

        <form onSubmit={onSubmit} className='flex flex-col w-full max-w-sm relative'>
          <div className='w-full my-4'>
            <div className='text-xxs text-gray-500 uppercase'>Email</div>
            <input
              required
              autoFocus
              type='email'
              placeholder='email@address.com'
              onChange={e => {
                setEmail(e.target.value)
                setError('')
              }}
              className={`w-full bg-transparent border-b border-gray-800 text-xs px-px py-3 focus:outline-none focus:border-b focus:border-gray-200 placeholder:italic ${error ? 'border-pink-500/60' : ''}`}
            />
          </div>
          <div className='w-full my-4'>
            <label for='password' className='text-xxs text-gray-500 uppercase'>Password</label>
            <input
              required
              name='password'
              type='password'
              placeholder='enter your password'
              onChange={e => {
                setPassword(e.target.value)
                setError('')
              }}
              className={`w-full bg-transparent border-b border-gray-800 text-xs px-px py-3 focus:outline-none focus:border-b focus:ring-gray-200 placeholder:italic ${error ? 'border-pink-500/60' : ''}`}
            />
          </div>
          <button disabled={!email || !password} className='border border-violet-300 hover:border-violet-100 my-2 text-xs px-4 py-3 rounded-lg disabled:pointer-events-none text-violet-100 disabled:opacity-30'>
            Login
          </button>
          {error && <p className='absolute -bottom-3.5 w-full mx-auto text-xs text-pink-400 text-center'>{error}</p>}
        </form>
      </div>
    </div>
  )
}
