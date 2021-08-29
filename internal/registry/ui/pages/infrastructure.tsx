import Head from 'next/head'
import dayjs from 'dayjs'
import { Fragment } from 'react'
import relativeTime from 'dayjs/plugin/relativeTime'
import { Popover, Transition } from '@headlessui/react'
import useSWR from 'swr'

import { DestinationsApi, Destination, ApikeysApi } from '../api'
import Dashboard from '../layouts/Dashboard'

dayjs.extend(relativeTime)

function Table ({ destinations }: { destinations: Destination[] | undefined }) {
  return (
    <div className="flex flex-col">
      <div className="align-middle inline-block">
        <table className="w-full divide-y divide-gray-200">
          <tbody className="divide-y divide-gray-200">
            {destinations?.map(d => (
              <tr key={d.id} className="text-sm border-gray-200 group">
                <td className="pl-6 pr-1 py-5 whitespace-nowrap text-black">
                  <div>{d.name}</div>
                  <div className="flex items-center text-xs text-gray-500 mt-1">
                    <div className="bg-blue-600 rounded-full w-2 h-2 mr-1"></div> Connected
                  </div>
                </td>
                <td className="pr-4 py-5 text-right whitespace-nowrap text-gray-500">Added {dayjs(Number(d.created) * 1000).fromNow()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

export default function Index () {
  const { isValidating, data: destinations } = useSWR(
    'destinations',
    () => new DestinationsApi().listDestinations(),
    {
      refreshInterval: 5000,
    }
   )

   const { data: apiKey } = useSWR(
    'apiKeys',
    () => new ApikeysApi().listApikeys().then(apikeys => apikeys[0] || null)
  )

  return (
    <Dashboard>
      <Head>
        <title>Infrastructure – Infra</title>
        <meta property="og:title" content="Infrastructure - Infra" key="title" />
      </Head>
      <div className="flex flex-col bg-white rounded-lg shadow mt-8">
        <div className="flex justify-between items-center pl-6 pr-4 border-b">
          <h1 className="text-md font-semibold text-black py-4">Clusters</h1>
          <Popover className="relative">
            {({ open }) => (
              <>
                <Popover.Button
                  className="inline-flex items-center px-3 py-2 border border-transparent text-sm leading-4 font-medium rounded-md shadow-sm text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-600"
                >
                  Add Cluster
                </Popover.Button>
                <Transition
                  show={open}
                  as={Fragment}
                  enter="transition ease-out duration-100"
                  enterFrom="transform opacity-0 scale-95"
                  enterTo="transform opacity-100 scale-100"
                  leave="transition ease-in duration-75"
                  leaveFrom="transform opacity-100 scale-100"
                  leaveTo="transform opacity-0 scale-95"
                >
                  <Popover.Panel
                    static
                    className="absolute z-10 right-0 transform mt-2"
                  >
                    <div className="rounded-xl shadow-2xl bg-gray-900 overflow-hidden">
                      <div className="bg-gray-800 uppercase text-gray-400 font-bold px-6 select-none py-4 text-xs tracking-wider">
                        Add Kubernetes Cluster
                      </div>
                      <div className="flex-1 relative gap-6 px-6 py-5 font-mono text-sm text-white leading-6 overflow-x-scroll whitespace-nowrap">
                        helm repo add infrahq https://helm.infrahq.com<br />
                        helm install infra-engine infrahq/engine \<br />
                          &nbsp;&nbsp;--set registry={process.browser && window.location.host} \<br />
                          &nbsp;&nbsp;--set apiKey={apiKey?.key}
                      </div>
                    </div>
                  </Popover.Panel>
                </Transition>
              </>
            )}
          </Popover>
        </div>
        {isValidating ? (
          <div className="flex-1 flex justify-center items-center py-8 text-gray-600 stroke-current">
            <svg width="34" height="34" viewBox="0 0 34 34" fill="none" xmlns="http://www.w3.org/2000/svg" className="animate-spin w-6 h-6">
              <path d="M33 17C33 8.16344 25.8366 1 17 1C8.16344 1 1 8.16344 1 17C1 25.8366 8.16344 33 17 33" strokeWidth="2"/>
            </svg>
          </div>
        ) : destinations?.length === 0 ? (
          <div className="text-center pb-24 pt-20">
            <h1 className="text-xl font-medium  text-black">No clusters</h1>
            <h4 className="mt-3 text-sm text-gray-700">Waiting for your first Kubernetes cluster...</h4>
          </div>
        ) : (
          <Table destinations={destinations} />
        )}
      </div>
    </Dashboard>
  )
}
