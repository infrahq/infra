import * as React from 'react'
import {
  ExclamationCircleIcon,
  InformationCircleIcon,
  CheckCircleIcon,
} from '@heroicons/react/24/outline'

const icons = {
  warning: ExclamationCircleIcon,
  info: InformationCircleIcon,
  success: CheckCircleIcon,
}

export default function Callout({ type, children }) {
  const Icon = icons[type]

  const styles = {
    warning: 'bg-yellow-400/10 border-yellow-400/30',
    info: 'bg-blue-600/5 border-blue-600/10',
    success: 'bg-emerald-500/10 border-emerald-400/30',
  }

  const iconstyle = {
    warning: 'text-yellow-500',
    info: 'text-blue-600/75',
    success: 'text-emerald-400',
  }

  return (
    <div
      className={`flex items-center rounded-xl border px-4 ${styles[type]} mt-4 mb-8`}
    >
      <Icon
        className={`mt-[20px] mr-4 h-5 w-5 flex-none self-start stroke-current ${iconstyle[type]}`}
      />
      <div className='overflow-hidden prose-p:text-base prose-p:leading-tight first-letter:prose-p:my-3'>
        {children}
      </div>
    </div>
  )
}
