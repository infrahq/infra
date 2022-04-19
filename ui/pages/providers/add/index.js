import Link from 'next/link'
import Head from 'next/head'

import FullscreenModal from '../../../components/modals/fullscreen'

const providers = [{
  name: 'Okta',
  icon: '/okta.svg',
  available: true
}, {
  name: 'Google',
  icon: '/google.svg'
}, {
  name: 'Azure Active Directory',
  icon: '/azure.svg'
}, {
  name: 'GitHub',
  icon: '/github.svg'
}, {
  name: 'GitLab',
  icon: '/gitlab.svg'
}, {
  name: 'OpenID',
  icon: '/openid.svg'
}]

function Provider ({ icon, name, available }) {
  return (
    <div key={name} className={`rounded-xl px-6 py-4 flex items-center bg-purple-100/5 ${available ? 'hover:bg-purple-100/10 cursor-pointer' : 'opacity-50 grayscale select-none'}`}>
      <img className='flex-none w-8 mr-4' src={icon} />
      <div>
        <h3 className='flex-1 font-medium'>{name}</h3>
        <h4 className='text-sm text-gray-400'>{available ? 'Identity Provider' : 'Coming Soon'}</h4>
      </div>
    </div>
  )
}

export default function () {
  return (
    <FullscreenModal closeHref='/providers'>
      <Head>
        <title>Add Identity Provider</title>
      </Head>
      <div className='flex flex-col mb-10'>
        <div className='flex my-4 bg-gradient-to-br from-violet-400/30 to-pink-200/30 items-center justify-center rounded-full mx-auto'>
          <div className='flex bg-black items-center justify-center rounded-full w-16 h-16 m-0.5'>
            <img className='w-8 h-8' src='/providers-color.svg' />
          </div>
        </div>
        <h1 className='text-white text-lg font-bold mb-1 text-center'>Add Identity Provider</h1>
        <h2 className='text-gray-300 mb-4 text-sm max-w-xs mx-auto text-center'>Select an identity provider to continue.</h2>
        <div className='grid grid-cols-3 gap-1 my-8'>
          {providers.map(p => (
            p.available
              ? (
                <Link href={`/providers/add/${p.name.toLowerCase()}`}>
                  <a>
                    <Provider {...p} />
                  </a>
                </Link>
                )
              : (
                <Provider {...p} />
                )
          ))}
        </div>
      </div>
    </FullscreenModal>
  )
}
