import Head from 'next/head'
import Router from 'next/router'
import { useState, useEffect } from 'react'
import useSWR from 'swr'

import Fullscreen from '../../../components/modals/fullscreen'
import HeaderIcon from '../../../components/dashboard/headerIcon'
import InputDropdown from '../../../components/inputDropdown'

const CommandInput = ({ enabledCommandInput, accessKey, currentDestinationName }) => {
  const server = window.location.host
  const isHttps = window.location.origin.includes('https')
  const defaultValue = `helm install infra-connector infrahq/infra \\
  --set connector.config.accessKey=${accessKey} \\
  --set connector.config.server=${server} \\
  --set connector.config.name=${currentDestinationName}`

  const commandValue = isHttps
    ? defaultValue
    : defaultValue + ` \\
  --set connector.config.skipTLSVerify=true`

  const value = enabledCommandInput ? commandValue : ''

  return (
    <div className='border-2 border-gray-800 rounded-lg shadow-sm overflow-hidden my-5'>
      <textarea
        rows={5}
        name='commandInput'
        id='commandInput'
        className='block w-full py-3 pl-3 border-0 resize-none sm:text-sm bg-black focus:outline-none whitespace-pre'
        value={value}
        readOnly
      />
    </div>
  )
}

export default function () {
  const [connected, setConnected] = useState(false)
  const [enabledCommandInput, setEnabledCommandInput] = useState(false)
  const [name, setName] = useState('')
  const [connectorFullName, setConnectorFullName] = useState('')
  const [numDestinations, setNumDestinations] = useState(0)

  const [accessKey, setAccessKey] = useState('')
  const [currentDestinationName, setCurrentDestinationName] = useState('')

  const { data: destinations } = useSWR('/v1/destinations')

  useEffect(() => {
    const handleDestinationConnection = () => {
      if (accessKey && name.length > 0) {
        fetch(`/v1/destinations?name=${connectorFullName}`)
          .then((response) => response.json())
          .then((data) => {
            if (!connected) {
              if (data.length === numDestinations) {
                pollingTimeout = setTimeout(handleDestinationConnection, 5000)
              } else {
                setConnected(true)
                clearTimeout(pollingTimeout)
              }
            }
          })
          .catch((error) => {
            console.log(error)
            clearTimeout(pollingTimeout)
          })
      }
    }

    let pollingTimeout = setTimeout(handleDestinationConnection, 5000)

    return () => {
      clearTimeout(pollingTimeout)
    }
  }, [accessKey])

  const handleFinished = () => {
    Router.replace('/destinations')
  }

  const handleNext = () => {
    const type = 'kubernetes'
    const destinationName = type + '.' + name

    setCurrentDestinationName(name)
    setEnabledCommandInput(name.length > 0)
    setConnectorFullName(destinationName)
    setNumDestinations(destinations.filter((item) => item.name === name).length)

    fetch('/v1/identities?name=connector')
      .then((response) => response.json())
      .then((data) => {
        const { id } = data[0]
        const keyName = name + '-' + [...Array(10)].map(() => (~~(Math.random() * 36)).toString(36)).join('')

        return { identityID: id, name: keyName, ttl: '87600h', extensionDeadline: '720h' }
      })
      .then((config) => {
        return fetch('/v1/access-keys', {
          method: 'POST',
          body: JSON.stringify(config)
        })
      })
      .then((response) => response.json())
      .then((accessKeyInfo) => {
        setAccessKey(accessKeyInfo.accessKey)
      })
      .catch((error) => console.log(error))
  }

  const handleKeyDownEvent = (key) => {
    if (key === 'Enter' && name.length > 0) {
      handleNext()
    }
  }

  return (
    <Fullscreen closeHref='/destinations' verticalCenteredContent={false}>
      <Head>
        <title>Infra - Destinations</title>
      </Head>
      <div className='flex flex-col mb-10 w-full max-w-md'>
        <HeaderIcon iconPath='/destinations-color.svg' position='center' />
        <h1 className='text-xl font-bold tracking-tight text-center'>Connect a Kubernetes Cluster</h1>
        <h2 className='mt-3 mb-5 text-gray-500 text-center'>
          For more info on destinations, check out our <a className='text-cyan-400 underline' target='_blank' href='https://infrahq.com/docs/connectors/kubernetes' rel='noreferrer'>docs</a>
        </h2>
        <div className='flex gap-1 mb-5'>
          <div className='flex-1 w-full'>
            <InputDropdown
              type='text'
              value={name}
              placeholder='Name your cluster'
              hasDropdownSelection={false}
              handleInputChange={e => setName(e.target.value)}
              handleKeyDown={(e) => handleKeyDownEvent(e.key)}
            />
          </div>
          <button
            onClick={() => handleNext()}
            disabled={name.length === 0}
            type='button'
            className='bg-gradient-to-tr from-indigo-300 to-pink-100 hover:from-indigo-200 hover:to-pink-50 p-0.5 mx-auto rounded-full'
          >
            <div className='bg-black flex items-center text-sm px-12 py-3 rounded-full'>
              Next
            </div>
          </button>
        </div>
        <h2 className='text-gray-500 text-center px-2'>
          After you name your cluster, run the output of this command to connect.
        </h2>
        <CommandInput
          enabledCommandInput={enabledCommandInput}
          accessKey={accessKey}
          currentDestinationName={currentDestinationName}
        />
        <h2 className='text-gray-500 text-center px-2 mb-2'>
          Once you have successfully installed Infra, we will be able to detect the connection.
        </h2>
        {enabledCommandInput &&
          <div className='border-2 border-dashed border-pink-300 opacity-60 rounded-lg shadow-sm overflow-hidden my-5 px-5 py-3'>
            <div className='flex items-center justify-center p-0.5 w-full'>
              <img className={connected ? 'w-8 h-8 animate-pulse' : 'w-8 h-8 animate-spin-fast'} src={connected ? '/connected-icon.svg' : '/connecting-spinner.svg'} />
              <p className='text-pink-500 text-sm px-2 py-3'>{connected ? 'Connected!' : 'Connecting...'}</p>
            </div>
          </div>}
        {connected &&
          <button
            onClick={() => handleFinished()}
            disabled={name.length === 0}
            type='button'
            className='bg-gradient-to-tr from-indigo-300 to-pink-100 hover:from-indigo-200 hover:to-pink-50 rounded-full p-0.5 w-full mt-6 text-center'
          >
            <div className='bg-black rounded-full tracking-tight text-sm px-6 py-3'>
              Finished
            </div>
          </button>}
      </div>
    </Fullscreen>
  )
}
