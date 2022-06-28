import * as React from 'react'
import {
  ExclamationIcon,
  InformationCircleIcon,
  CheckCircleIcon,
} from '@heroicons/react/outline'

const icons = {
  warning: ExclamationIcon,
  info: InformationCircleIcon,
  success: CheckCircleIcon,
}

export default function Callout({ type, children }) {
  const Icon = icons[type]

  const styles = {
    warning: 'bg-amber-400/10 border-amber-400/20',
    info: 'bg-indigo-400/10 border-indigo-300/20',
    success: 'bg-teal-400/10 border-teal-400/20',
  }

  const iconstyle = {
    warning: 'text-yellow-200',
    info: 'text-indigo-200',
    success: 'text-teal-200',
  }

  return (
    <div
      className={`flex items-center rounded-xl border px-4 ${styles[type]} mt-6 mb-8`}
    >
      <Icon
        className={`mt-6 mr-4 h-5 w-5 flex-none self-start stroke-current ${iconstyle[type]}`}
      />
      <div className='overflow-hidden prose-p:text-base prose-p:leading-tight first-letter:prose-p:my-3'>
        {children}
      </div>
    </div>
  )
}
