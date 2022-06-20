import path from 'path'
import fs from 'fs'
import { imageSize } from 'image-size'
import Markdoc, { Tag } from '@markdoc/markdoc'

export default {
  nodes: {
    fence: {
      render: 'Code',
      attributes: {
        language: { type: String }
      }
    },
    heading: {
      render: 'Heading',
      children: ['inline'],
      attributes: {
        id: { type: String },
        className: { type: String },
        level: { type: Number, required: true, default: 1 }
      }
    },
    link: {
      render: 'Link',
      attributes: {
        href: {
          description: 'The path or URL to navigate to.',
          type: String,
          errorLevel: 'critical',
          required: true
        }
      },
      transform (node, config) {
        const attributes = node.transformAttributes(config)
        const children = node.transformChildren(config)

        if (attributes.href.startsWith('./') || attributes.href.startsWith('../')) {
          attributes.href = path.join(path.dirname(node.location.file), attributes.href?.replace('.md', ''))
        }

        return new Tag(this.render, attributes, children)
      }
    },
    image: {
      render: 'Image',
      attributes: {
        src: { type: String, required: true },
        alt: { type: String },
        width: { type: Number },
        height: { type: Number },
        layout: { type: String }
      },
      transform (node, config) {
        const attributes = node.transformAttributes(config)
        const filepath = path.join(process.cwd(), '..', node.location.file, '..', attributes.src)
        const destination = path.join(process.cwd(), 'public', node.location.file, '..', attributes.src)

        if (fs.statSync(filepath).isFile()) {
          const dirname = path.dirname(destination)
          fs.mkdirSync(dirname, { recursive: true })
          fs.copyFileSync(filepath, destination)

          const { width, height } = imageSize(filepath)

          attributes.width = width
          attributes.height = height
        } else {
          attributes.layout = 'fill'
        }

        attributes.src = path.join(node.location.file, '..', attributes.src)
        return new Tag(this.render, attributes)
      }
    }
  },
  tags: {
    tabs: {
      render: 'Tabs',
      attributes: {},
      transform (node, config) {
        const tabs = node
          .transformChildren(config)
          .filter(child => child && child.name === 'Tab')

        const labels = tabs
          .map(tab => (typeof tab === 'object' ? tab.attributes.label : null))

        return new Tag(this.render, { labels }, tabs)
      }
    },
    tab: {
      render: 'Tab',
      attributes: {
        label: {
          type: String
        }
      }
    },
    callout: {
      render: 'Callout',
      description: 'Display the enclosed content in a callout box',
      children: ['paragraph', 'tag', 'list'],
      attributes: {
        type: {
          type: String,
          default: 'note',
          matches: ['warning', 'info', 'success'],
          errorLevel: 'critical',
          description: 'Controls the color and icon of the callout. Can be: "warning", "info", "success"'
        }
      }
    },
    youtube: {
      render: 'YouTube',
      attributes: {
        src: {
          type: String,
          required: true
        },
        title: {
          type: String,
          required: true
        },
        width: {
          type: String
        }
      }
    },
    partial: {
      selfClosing: true,
      attributes: {
        file: { type: String, render: false, required: true },
        variables: { type: Object, render: false }
      },
      transform (node, config) {
        const { partials = {} } = config
        const { file, variables } = node.attributes
        let partial = partials[file]

        if (!partial) {
          const filepath = path.join(process.cwd(), '..', node.location.file, '..', file)

          if (!fs.existsSync(filepath)) {
            return null
          }

          const contents = fs.readFileSync(filepath, 'utf-8')
          partial = Markdoc.parse(contents)
        }

        const scopedConfig = {
          ...config,
          variables: {
            ...config.variables,
            ...variables,
            '$$partial:filename': file
          }
        }

        const transformChildren = part => part.resolve(scopedConfig).transformChildren(scopedConfig)
        return Array.isArray(partial)
          ? partial.flatMap(transformChildren)
          : transformChildren(partial)
      }
    }
  }
}
