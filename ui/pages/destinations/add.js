import Head from 'next/head'
import Link from 'next/link'
import { useState, useEffect } from 'react'
import useSWR from 'swr'
import copy from 'copy-to-clipboard'

import Fullscreen from '../../components/layouts/fullscreen'

export default function DestinationsAdd () {
  const [accessKey, setAccessKey] = useState('')
  const [name, setName] = useState('')
  const [submitted, setSubmitted] = useState(false)
  const [connected, setConnected] = useState(false)
  const [copied, setCopied] = useState(false)

  async function checkConnection () {
    if (accessKey && name.length > 0) {
      const res = await fetch(`/v1/destinations?name=${name}`)
      const destinations = await res.json()
      if (destinations.length > 0) {
        setConnected(true)
      }
    }
  }

  useEffect(() => {
    let interval = setInterval(checkConnection, 5000)
    return () => {
      clearInterval(interval)
    }
  }, [accessKey])

  async function handleNext () {
    setConnected(false)
    let res = await fetch('/v1/identities?name=connector')
    const connectors = await res.json()
    const { id } = connectors[0]
    const keyName = name + '-' + [...Array(10)].map(() => (~~(Math.random() * 36)).toString(36)).join('')
    res = await fetch('/v1/access-keys', {
      method: 'POST',
      body: JSON.stringify({ userID: id, name: keyName, ttl: '87600h', extensionDeadline: '720h' })
    })
    const key = await res.json()
    setAccessKey(key.accessKey)
  }

  const server = window.location.host
  const command = `helm install infra-connector infrahq/infra \\
    --set connector.config.accessKey=${accessKey} \\
    --set connector.config.server=${server} \\
    --set connector.config.name=${name} \\
    --set connector.config.skipTLSVerify=true`

  return (
    <div>
      <Head>
        <title>Add Infrastructure - Infra</title>
      </Head>
      <header className='flex flex-row px-4 pt-5 pb-6 items-center'>
        <img src='/destinations.svg' className='w-6 h-6 mr-2 mt-0.5' />
        <h1 className='text-xs capitalize'>Connect infrastructure</h1>
      </header>
      <form
        onSubmit={e => {
          e.preventDefault()
          setSubmitted(true)
          handleNext()
          return false
        }}
        className='flex space-x-2 px-4 mb-10'
      >
        <div className='flex-1'>
          <label className='text-xxs text-gray-400 uppercase'>Cluster Name</label>
          <input
            autoFocus
            required
            placeholder='provide a cluster name'
            value={name}
            onChange={e => setName(e.target.value)}
            disabled={submitted}
            className='w-full bg-transparent border-b border-gray-800 text-xxs px-px py-2 focus:outline-none focus:border-b focus:border-gray-200 placeholder:italic disabled:opacity-10'
          />
        </div>
        <button
          className='flex-none border border-violet-300 rounded-md text-violet-100 self-end text-xs px-4 py-2 disabled:opacity-10'
          disabled={submitted}
        >
          Next
        </button>
      </form>
      <section className={`flex flex-col my-2 ${submitted ? '' : 'opacity-10 pointer-events-none'}`}>
        <h2 className='px-4 text-xs mb-2 text-gray-100'>Run this command on your Kubernetes cluster:</h2>
        <pre className={`text-xs p-4 min-h-[120px] text-gray-300 bg-gray-900 ${submitted ? 'overflow-auto' : 'overflow-hidden'}`}>
          {submitted ? command : ''}
        </pre>
        <button
          className='self-end text-xxs text-violet-200 mt-2 mb-3 py-2 px-3 mr-2 font-medium uppercase disabled:text-gray-500'
          disabled={copied}
          onClick={() => {
            copy(command)
            setCopied(true)
            setTimeout(() => setCopied(false), 2000)
          }}
        >
          {copied ? '✓ Copied' : 'Copy command'}
        </button>
        <p className='text-gray-500 text-xs px-4'>Your cluster will be detected automatically.<br />This may take a few minutes.</p>
        {connected ? (
          <footer className='flex justify-between px-4 my-4 mr-3 items-center'>
            <h3 className='text-xs text-gray-200'>✓ Connected</h3>
            <Link href='/destinations'>
              <a
                className='flex-none border border-violet-300 rounded-md text-violet-100 self-end text-xs px-4 py-2 disabled:opacity-20'
                disabled={submitted}
              >
                Finish
              </a>
            </Link>
          </footer>
        ) : (
        <footer className='flex items-center px-4 my-5'>
          <h3 className='text-xs mr-3 text-gray-200'>Waiting for connection</h3>
          {submitted && <span className='animate-[ping_1.25s_ease-in-out_infinite] flex-none inline-flex h-2 w-2 rounded-full border border-white opacity-75' />}
        </footer>
        )}
      </section>
    </div>
  )
}

DestinationsAdd.layout = page =>
  <Fullscreen closeHref='/destinations'>
    {page}
  </Fullscreen>
