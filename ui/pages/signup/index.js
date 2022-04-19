import { useState } from 'react'
import axios from 'axios'
import Link from 'next/link'
import { useSWRConfig } from 'swr'
import { useRouter } from 'next/router'

function Welcome ({ signup }) {
  return (
    <>
      <h2 className='text-center my-4'>Infra has been successfully installed.</h2>
      <button className='w-full my-3 bg-zinc-500/20 hover:bg-gray-500/25 py-2.5 rounded-md text-white text-md hover:cursor-pointer' onClick={() => signup()}>Next</button>
    </>
  )
}

function Finish ({ accessKey }) {
  const { mutate } = useSWRConfig()
  const router = useRouter()

  async function login () {
    // log in
    const res = await fetch('/v1/login', {
      method: 'POST',
      body: JSON.stringify({ accessKey })
    })
    const { name } = await res.json()
    mutate('/v1/introspect', { optimisticData: { name } })
    mutate('/v1/setup', { optimisticData: { required: false } })
    router.replace('/')
  }

  return (
    <>
      <h2 className='text-center text-gray-400 mt-12'>This is your admin key. Please back it up in a safe place.</h2>
      <h3 className='font-mono text-2xl text-white mt-4 mb-12'>{accessKey}</h3>
      <Link href='/'>
        <a className='self-stretch'>
          <button
            className='w-full my-3 bg-zinc-500/20 hover:bg-gray-500/25 py-2.5 rounded-md text-white text-md hover:cursor-pointer'
            onClick={() => login()}
          >
            Get Started
          </button>
        </a>
      </Link>
    </>
  )
}

export default function () {
  const [accessKey, setAccessKey] = useState('')

  async function signup () {
    try {
      const { data: { accessKey } } = await axios.post('/v1/setup')

      setAccessKey(accessKey)
    } catch (e) {
      console.log(e)
    }
  }

  return (
    <div className='flex flex-col justify-center items-center h-full w-full max-w-md mx-auto mb-48'>
      <img className='text-white w-10 h-10' src='/infra-icon.svg' />
      <h1 className='my-5 text-3xl font-light tracking-tight'>Welcome to Infra</h1>
      {accessKey ? <Finish accessKey={accessKey} /> : <Welcome signup={signup} />}
    </div>
  )
}
