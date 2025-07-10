//
//  test_file_space.swift
//  test-project
//
//  Created by chris on 10/24/23.
//

import ArgumentParser
import Foundation
#if DEBUG
import AppKit
#endif

@main
struct Application: AsyncParsableCommand {
    static let configuration = CommandConfiguration(
        commandName: "my-command",
        shouldDisplay: true,
        subcommands: [
            Version.self,
            Create.self,
            Run.self,
        ]
    )

    @Option(parsing: .remaining)
    var uncategorizedFlags: [String] = []

    public static func main() async throws {
        #if !DEBUG
        setbuf(stdout, nil)
        do {
            var main = try parseAsRoot()
            if var asyncMain = main as? AsyncParsableCommand {
                try await asyncMain.run()
            } else {
                try main.run()
            }
        } catch {
            exit(withError: error)
        }
        #else
        NSApplication.shared.run()
        #endif
    }
}
