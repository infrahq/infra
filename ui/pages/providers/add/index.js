import Link from 'next/link'

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

export default function () {
  return (
    <FullscreenModal closeHref='/providers'>
      <div className='mb-10'>
        <h1 className='text-3xl font-bold tracking-tight text-center'>Add Identity Provider</h1>
        <h2 className='mt-2 mb-10 text-gray-300 text-center'>Select an identity provider to continue</h2>
        <div className='grid grid-cols-3 gap-1'>
          {providers.map(p => (
            <div key={p.name} className={`rounded-xl px-6 py-4 bg-zinc-900 ${p.available ? 'hover:bg-zinc-800 cursor-pointer' : 'opacity-50 grayscale select-none'}`}>
              {p.available
                ? (
                  <Link href={`/providers/add/${p.name.toLowerCase()}`}>
                    <a className='flex items-center'>
                      <img className='flex-none h-4 mr-4' src={p.icon} />
                      <div>
                        <h3 className='flex-1 font-medium'>{p.name}</h3>
                        <h4 className='text-sm text-gray-400'>{p.available ? 'Identity Provider' : 'Coming Soon'}</h4>
                      </div>
                    </a>
                  </Link>
                  )
                : (
                  <div className='flex items-center'>
                    <img className='flex-none h-4 mr-4' src={p.icon} />
                    <div>
                      <h3 className='flex-1 font-medium'>{p.name}</h3>
                      <h4 className='text-sm text-gray-400'>{p.available ? 'Identity Provider' : 'Coming Soon'}</h4>
                    </div>
                  </div>
                  )}
            </div>
          ))}
        </div>
      </div>
    </FullscreenModal>
  )
}
